package gitservice_test

import (
	"context"
	"testing"

	"github.com/circleous/gitseer/pkg/gitservice"
)

func TestListOrg(t *testing.T) {
	ctx := context.Background()
	gs := gitservice.NewGithubClient(ctx)
	org := "kolatif"
	members, err := gs.ListOrgUsers(ctx, org)
	if members == nil || err != nil {
		t.Fatalf("Error fetching users from org %s, %v", org, err.Error())
	}
	// TODO: check for members content
}

func TestListRepo(t *testing.T) {
	ctx := context.Background()
	gs := gitservice.NewGithubClient(ctx)
	user := "circleous"
	repos, err := gs.ListUserRepositories(ctx, user, &gitservice.ListRepositoriesOptions{
		WithFork: false,
	})
	if repos == nil || err != nil {
		t.Fatalf("Error fetching repos from user %s, %v", user, err.Error())
	}
	// TODO: check for repos content
}
