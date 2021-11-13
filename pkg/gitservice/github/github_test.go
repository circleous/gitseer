package github_test

import (
	"context"
	"testing"

	"github.com/circleous/gitseer/pkg/git"
	"github.com/circleous/gitseer/pkg/gitservice/github"
)

func TestListOrg(t *testing.T) {
	ctx := context.Background()
	gs := github.NewGithubClient(ctx)
	org := "traveloka"
	members, err := gs.ListOrgUsers(ctx, org)
	if members == nil || err != nil {
		t.Fatalf("Error fetching users from org %s, %v", org, err.Error())
	}
	// TODO: check for members content
}

func TestListRepo(t *testing.T) {
	ctx := context.Background()
	gs := github.NewGithubClient(ctx)
	user := "circleous"
	repos, err := gs.ListUserRepositories(ctx, user,
		&git.ListRepositoriesOptions{
			WithFork: false,
		})
	if repos == nil || err != nil {
		t.Fatalf("Error fetching repos from user %s, %v", user, err.Error())
	}
	// TODO: check for repos content
}
