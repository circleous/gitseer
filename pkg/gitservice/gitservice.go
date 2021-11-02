package gitservice

import (
	"context"
)

type ListRepositoriesOptions struct {
	// WithFork decides if forked repo should be included in the list
	WithFork bool
}

type GitServiceRepositories struct {
	Name string
	URL  string
}

type GitService interface {
	// ListOrgUsers return all users joined the organization
	ListOrgUsers(ctx context.Context, org string) ([]string, error)
	// ListUserRepositories return all repositorises from a user
	ListUserRepositories(ctx context.Context, user string, opt ListRepositoriesOptions) ([]*GitServiceRepositories, error)
}
