package gitservice

import (
	"context"
	"sync"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

type GithubService struct {
	client *github.Client
}

func NewGithubClient(ctx context.Context) *GithubService {
	client := github.NewClient(nil)

	return &GithubService{
		client: client,
	}
}

func NewGithubClientWithToken(ctx context.Context, token string) *GithubService {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &GithubService{
		client: client,
	}
}

func (gs *GithubService) ListOrgUsers(ctx context.Context, org string) ([]string, error) {
	var m sync.Mutex
	var wg sync.WaitGroup

	gitUsers, resp, err := gs.client.Organizations.ListMembers(ctx, org, &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, err
	}

	users := make([]string, len(gitUsers))
	for _, gitUser := range gitUsers {
		users = append(users, *gitUser.Login)
	}

	for page := resp.NextPage; page <= resp.LastPage; page++ {
		wg.Add(1)
		go func(m *sync.Mutex, wg *sync.WaitGroup, page int) {
			gitUsers, _, err := gs.client.Organizations.ListMembers(ctx, org, &github.ListMembersOptions{
				ListOptions: github.ListOptions{PerPage: 100, Page: page},
			})
			if err != nil {
				return
			}
			defer wg.Done()

			// rather than switch mutex every append(), better to switch one time
			// so that it wont spent so much time on context switches
			m.Lock()
			for _, gitUser := range gitUsers {
				users = append(users, *gitUser.Login)
			}
			m.Unlock()
		}(&m, &wg, page)
	}

	wg.Wait()

	return users, nil
}

func (gs *GithubService) ListUserRepositories(ctx context.Context, user string, opt *ListRepositoriesOptions) ([]*GitServiceRepositories, error) {
	var m sync.Mutex
	var wg sync.WaitGroup

	gitRepos, resp, err := gs.client.Repositories.List(ctx, user, &github.RepositoryListOptions{
		Visibility:  "public",
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, err
	}

	repos := make([]*GitServiceRepositories, len(gitRepos))
	for _, gitRepo := range gitRepos {
		if *gitRepo.Fork && !opt.WithFork {
			continue
		}
		repos = append(repos, &GitServiceRepositories{
			Name: *gitRepo.Name,
			URL:  *gitRepo.CloneURL,
		})
	}

	for page := resp.NextPage; page <= resp.LastPage; page++ {
		wg.Add(1)
		go func(m *sync.Mutex, wg *sync.WaitGroup, page int) {
			gitRepos, _, err := gs.client.Repositories.List(ctx, user, &github.RepositoryListOptions{
				ListOptions: github.ListOptions{PerPage: 100, Page: page},
			})
			// TODO: should log or better yet
			if err != nil {
				return
			}
			defer wg.Done()

			// rather than switch mutex every append(), better to switch one time
			// so that it wont spent so much time on context switches
			m.Lock()
			for _, gitRepo := range gitRepos {
				if *gitRepo.Fork && !opt.WithFork {
					continue
				}
				repos = append(repos, &GitServiceRepositories{
					Name: *gitRepo.Name,
					URL:  *gitRepo.CloneURL,
				})
			}
			m.Unlock()
		}(&m, &wg, page)
	}

	wg.Wait()

	return repos, nil
}
