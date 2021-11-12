package github

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/rs/zerolog/log"
)

const writeDelay = 1 * time.Second

// rateLimitTransport implements GitHub's best practices
// for avoiding rate limits
// https://developer.github.com/v3/guides/best-practices-for-integrators/#dealing-with-abuse-rate-limits
type rateLimitTransport struct {
	transport        http.RoundTripper
	delayNextRequest bool
	responseBody     []byte

	m sync.Mutex
}

func (rlt *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Make requests for a single user or client ID serially
	// This is also necessary for safely saving
	// and restoring bodies between retries below
	rlt.lock(req)

	// If you're making a large number of POST, PATCH, PUT, or DELETE requests
	// for a single user or client ID, wait at least one second between each request.
	if rlt.delayNextRequest {
		log.Debug().Msgf("Sleeping %s between write operations", writeDelay)
		time.Sleep(writeDelay)
	}

	rlt.delayNextRequest = isWriteMethod(req.Method)

	resp, err := rlt.transport.RoundTrip(req)
	if err != nil {
		rlt.unlock(req)
		return resp, err
	}

	// Make response body accessible for retries & debugging
	// (work around bug in GitHub SDK)
	// See https://github.com/google/go-github/pull/986
	r1, r2, err := drainBody(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = r1
	ghErr := github.CheckResponse(resp)
	resp.Body = r2

	// When you have been limited, use the Retry-After response header to slow down.
	if arlErr, ok := ghErr.(*github.AbuseRateLimitError); ok {
		rlt.delayNextRequest = false
		retryAfter := arlErr.GetRetryAfter()
		log.Debug().Msgf("Abuse detection mechanism triggered, sleeping for %s before retrying",
			retryAfter)
		time.Sleep(retryAfter)
		rlt.unlock(req)
		return rlt.RoundTrip(req)
	}

	if rlErr, ok := ghErr.(*github.RateLimitError); ok {
		rlt.delayNextRequest = false
		retryAfter := rlErr.Rate.Reset.Sub(time.Now())
		log.Debug().Msgf("Rate limit %d reached, sleeping for %s (until %s) before retrying",
			rlErr.Rate.Limit, retryAfter, time.Now().Add(retryAfter))
		time.Sleep(retryAfter)
		rlt.unlock(req)
		return rlt.RoundTrip(req)
	}

	rlt.unlock(req)

	return resp, nil
}

func (rlt *rateLimitTransport) lock(req *http.Request) {
	rlt.m.Lock()
}

func (rlt *rateLimitTransport) unlock(req *http.Request) {
	rlt.m.Unlock()
}

// newRateLimitTransport creates new roundtripper rate limiter
func newRateLimitTransport(rt http.RoundTripper) *rateLimitTransport {
	return &rateLimitTransport{transport: rt}
}

// drainBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func isWriteMethod(method string) bool {
	switch method {
	case "POST", "PATCH", "PUT", "DELETE":
		return true
	}
	return false
}
