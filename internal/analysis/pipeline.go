package analysis

import (
	"context"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"

	cgit "github.com/circleous/gitseer/pkg/git"
)

func (a *analysis) processOrganizations(ctx context.Context) {
	listRepoOpt := &cgit.ListRepositoriesOptions{
		WithFork: a.config.WithFork,
	}

	// Blocking request ahead, need more research for avoiding secondary rate limit
	for _, org := range a.config.Organizations {
		log.Debug().Str("organization", org.Name).Msg("processing org")

		if !org.ExpandRepo && !org.ExpandUser {
			log.Info().Str("organization", org.Name).Str("type", org.Type).Msg("atleast one of expand_user or expand_repo needs to be true")
			continue
		}

		if org.ExpandUser {
			u, err := a.gs.ListOrgUsers(ctx, org.Name, org.Type)
			if err != nil {
				log.Error().Err(err).Str("organization", org.Name).Str("type", org.Type).Msg("failed to fetch organization users")
			} else if len(u) > 0 {
				a.users = append(a.users, u...)
			}
		}

		if org.ExpandRepo {
			r, err := a.gs.ListOrgRepositories(ctx, org.Name, org.Type, listRepoOpt)
			if err != nil {
				log.Error().Err(err).Str("organization", org.Name).Str("type", org.Type).Msg("failed to fetch organization repositories")
			} else if len(r) > 0 {
				a.repositories = append(a.repositories, r...)
			}
		}
	}
}

func (a *analysis) processUsers(ctx context.Context) {
	listRepoOpt := &cgit.ListRepositoriesOptions{
		WithFork: a.config.WithFork,
	}

	// Blocking request ahead, need more research for avoiding secondary rate limit
	for _, user := range a.users {
		log.Debug().Str("user", user.Name).Msg("processing user")
		r, err := a.gs.ListUserRepositories(ctx, user.Name, user.Type, listRepoOpt)
		log.Debug().Str("user", user.Name).Msgf("got repo %d", len(r))
		if err != nil {
			log.Error().Err(err).Str("user", user.Name).Str("type", user.Type).Msg("failed to fetch user repositories")
		} else if len(r) > 0 {
			a.repositories = append(a.repositories, r...)
		}
	}
}

func (a *analysis) processRepositories(ctx context.Context) {
	sem := semaphore.NewWeighted(int64(a.config.MaxWorker))

	config := *a.config // copy

	for _, repo := range a.repositories {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Error().Err(err).Msg("Failed to acquire semaphore")
			break
		}

		repo := repo // copy

		// TODO: benchmark this path, currently we only use one goroutine per repository
		go func() {
			defer sem.Release(1)
			processRepository(repo, config)
		}()
	}

	if err := sem.Acquire(ctx, int64(a.config.MaxWorker)); err != nil {
		log.Error().Err(err).Msg("Failed to acquire semaphore")
	}
}

// Runner run the overall analysis pipeline
func (a *analysis) Runner() {
	// should we add timeout?
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// do we need to stop when error occurs?
	// a.processOrganizations(ctx)
	// a.processUsers(ctx)

	a.repositories = append(a.repositories, cgit.Repository{
		URL:          "https://github.com/circleous/dotfiles.git",
		Name:         "circleous/dotfiles",
		LatestCommit: "",
	})
	a.processRepositories(ctx)

	log.Debug().Msgf("%v", a.users)
	log.Debug().Msgf("%v", a.repositories)
}
