package github

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
	"golang.org/x/sync/semaphore"

	"github.com/circleous/gitseer/pkg/git"
)

type githubService struct {
	client    *github.Client
	maxWorker int
}

// Service exported interface for github service
type Service interface {
	// ListOrgUsers return all users joined the organization
	ListOrgUsers(ctx context.Context, org string) ([]git.User, error)
	// ListUserRepositories return all repositorises from a user, can be used for org
	ListUserRepositories(ctx context.Context, user string, opt *git.ListRepositoriesOptions) ([]git.Repository, error)
}

// NewGithubClient create plain new github api client without token, change max worker in
// context.Value with gitservice.MaxWorkerKey as key
func NewGithubClient(ctx context.Context) Service {
	var maxWorker int
	var ok bool

	hc := http.DefaultClient
	hc.Transport = newRateLimitTransport(http.DefaultTransport)
	client := github.NewClient(hc)

	if maxWorker, ok = ctx.Value(git.MaxWorkerKey).(int); !ok {
		maxWorker = 10 // fallback
	}

	return &githubService{
		client:    client,
		maxWorker: maxWorker,
	}
}

// NewGithubClientWithToken create new github api client with token
func NewGithubClientWithToken(ctx context.Context, token string) Service {
	var maxWorker int
	var ok bool

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	hc := oauth2.NewClient(ctx, ts)
	hc.Transport = newRateLimitTransport(hc.Transport)
	client := github.NewClient(hc)

	if maxWorker, ok = ctx.Value(git.MaxWorkerKey).(int); !ok {
		maxWorker = 10 // fallback
	}

	return &githubService{
		client:    client,
		maxWorker: maxWorker,
	}
}

// ListOrgUsers return all github users joined the organization
func (ghs *githubService) ListOrgUsers(ctx context.Context, org string) ([]git.User, error) {
	var m sync.Mutex
	var users []git.User

	sem := semaphore.NewWeighted(int64(ghs.maxWorker))

	gitUsers, resp, err := ghs.client.Organizations.ListMembers(ctx, org, &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, err
	}

	if len(gitUsers) == 0 {
		return nil, nil
	}

	for _, gitUser := range gitUsers {
		users = append(users, git.User{
			Name: *gitUser.Login,
			Type: git.GITHUB,
		})
	}

	for page := resp.NextPage; page < resp.LastPage; page++ {
		sem.Acquire(ctx, 1)
		page := page // copy
		go func() {
			gitUsers, _, err := ghs.client.Organizations.ListMembers(ctx, org, &github.ListMembersOptions{
				ListOptions: github.ListOptions{PerPage: 100, Page: page},
			})
			if err != nil {
				return
			}

			// rather than switch mutex every append(), better to switch one time
			// so that it wont spent so much time on context switches
			m.Lock()
			for _, gitUser := range gitUsers {
				users = append(users, git.User{
					Name: *gitUser.Login,
					Type: git.GITHUB,
				})
			}
			m.Unlock()

			sem.Release(1)
		}()
	}

	// wait
	if err := sem.Acquire(ctx, int64(ghs.maxWorker)); err != nil {
		return nil, err
	}

	return users, nil
}

// ListUserRepositories return all repositorises given user, when opt.WithFork is true, return will also includes
// forked repositories
func (ghs *githubService) ListUserRepositories(ctx context.Context, user string, opt *git.ListRepositoriesOptions) ([]git.Repository, error) {
	var m sync.Mutex
	var repos []git.Repository

	sem := semaphore.NewWeighted(int64(ghs.maxWorker))

	if opt != nil {
		opt = &git.DefaultListRepositoriesOpt
	}

	gitRepos, resp, err := ghs.client.Repositories.List(ctx, user, &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, err
	}

	if len(gitRepos) == 0 {
		return nil, nil
	}

	for _, gitRepo := range gitRepos {
		if *gitRepo.Fork && !opt.WithFork {
			continue
		}
		repos = append(repos, git.Repository{
			Name: *gitRepo.FullName,
			URL:  *gitRepo.CloneURL,
		})
	}

	errChan := make(chan error)

	for page := resp.NextPage; page < resp.LastPage; page++ {
		sem.Acquire(ctx, 1)
		page := page // copy
		go func() {
			gitRepos, _, err := ghs.client.Repositories.List(ctx, user, &github.RepositoryListOptions{
				ListOptions: github.ListOptions{PerPage: 100, Page: page},
			})
			// TODO: should log or better yet
			if err != nil {
				errChan <- err
				return
			}

			// rather than switch mutex every append(), better to switch one time
			// so that it wont spent so much time on context switches
			m.Lock()
			for _, gitRepo := range gitRepos {
				if *gitRepo.Fork && !opt.WithFork {
					continue
				}
				repos = append(repos, git.Repository{
					Name:         *gitRepo.FullName,
					URL:          *gitRepo.CloneURL,
					LatestCommit: "",
				})
			}
			m.Unlock()
			sem.Release(1)
		}()
	}
	close(errChan)

	// wait
	if err := sem.Acquire(ctx, int64(ghs.maxWorker)); err != nil {
		return nil, err
	}

	err = <-errChan
	if err != nil {
		return nil, err
	}

	return repos, nil
}
