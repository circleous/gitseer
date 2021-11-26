package gitservice

import (
	"context"
	"errors"

	"github.com/circleous/gitseer/pkg/git"
	"github.com/circleous/gitseer/pkg/gitservice/github"
)

var (
	defaultGitServiceOptions = Options{}
	// ErrInvalidServiceType errors for invlid service type
	ErrInvalidServiceType = errors.New("invalid service type")
)

// Service is the interface the gitservice
type Service interface {
	// ListOrgUsers return all users joined the organization
	// revive:disable-next-line:line-length-limit
	ListOrgUsers(ctx context.Context, serviceType string, org string) ([]git.User, error)
	// ListOrgRepositories return all repositorises in the organization
	// revive:disable-next-line:line-length-limit
	ListOrgRepositories(ctx context.Context, serviceType string, org string, opt *git.ListRepositoriesOptions) ([]git.Repository, error)
	// ListUserRepositories return all repositorises from a user
	// revive:disable-next-line:line-length-limit
	ListUserRepositories(ctx context.Context, serviceType string, user string, opt *git.ListRepositoriesOptions) ([]git.Repository, error)

	FindUserFuzzy(ctx context.Context, serviceType string, query string) ([]git.User, error)
}

// Options is the option struct when creating GitService
type Options struct {
	// GitlabToken string

	// GithubToken personal access token for accessing the github api
	GithubToken string
}

// GitService holds reference to multiple service
type gitService struct {
	// ghl *GitlabService
	ghs github.Service
}

// NewGitService a wrapper around the available git service api for listing
func NewGitService(ctx context.Context, opt *Options) Service {
	var githubSvc github.Service

	if opt == nil {
		opt = &defaultGitServiceOptions
	}

	if opt.GithubToken != "" {
		githubSvc = github.NewGithubClientWithToken(ctx, opt.GithubToken)
	} else {
		githubSvc = github.NewGithubClient(ctx)
	}

	return &gitService{
		ghs: githubSvc,
	}
}

// ListOrgUsers return all users joined the organization, valid serviceTypes are
// [github, gitlab]
func (gs *gitService) ListOrgUsers(ctx context.Context, serviceType string, org string) ([]git.User, error) {
	switch serviceType {
	case git.GITHUB:
		return gs.ghs.ListOrgUsers(ctx, org)
	}

	return nil, ErrInvalidServiceType
}

// ListOrgRepositories return all repositorises in the organization, valid
// serviceTypes are [github, gitlab], when opt.WithFork is true, return will
// also includes forked repositories
// revive:disable-next-line:line-length-limit
func (gs *gitService) ListOrgRepositories(ctx context.Context, serviceType string, org string, opt *git.ListRepositoriesOptions) ([]git.Repository, error) {
	switch serviceType {
	case git.GITHUB:
		return gs.ghs.ListUserRepositories(ctx, org, opt)
	}

	return nil, ErrInvalidServiceType
}

// ListUserRepositories return all repositorises given user, valid serviceTypes
// are [github, gitlab] when opt.WithFork is true, return will also includes
// forked repositories
// revive:disable-next-line:line-length-limit
func (gs *gitService) ListUserRepositories(ctx context.Context, serviceType string, user string, opt *git.ListRepositoriesOptions) ([]git.Repository, error) {
	switch serviceType {
	case git.GITHUB:
		return gs.ghs.ListUserRepositories(ctx, user, opt)
	}

	return nil, ErrInvalidServiceType
}

func (gs *gitService) FindUserFuzzy(ctx context.Context, serviceType string, query string) ([]git.User, error) {
	switch serviceType {
	case git.GITHUB:
		return gs.ghs.FindUserFuzzy(ctx, query)
	}

	return nil, ErrInvalidServiceType
}
