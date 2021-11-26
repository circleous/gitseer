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

	// Blocking request ahead, need more research for avoiding secondary rate
	// limit
	for _, org := range a.config.Organizations {
		log.Debug().Str("organization", org.Name).Msg("processing org")

		if !org.ExpandRepo && !org.ExpandUser {
			log.Info().Str("organization", org.Name).Str("type", org.Type).
				Msg("atleast one of expand_user / expand_repo needs to be true")
			continue
		}

		if org.ExpandUser {
			u, err := a.gs.ListOrgUsers(ctx, org.Type, org.Name)
			if err != nil {
				log.Error().Err(err).Str("organization", org.Name).
					Str("type", org.Type).
					Msg("failed to fetch organization users")
			} else if len(u) > 0 {
				a.users = append(a.users, u...)
			}
		}

		if org.ExpandUser {
			u, err := a.gs.FindUserFuzzy(ctx, org.Type, org.Name)
			if err != nil {
				log.Error().Err(err).Str("organization", org.Name).
					Str("type", org.Type).
					Msg("failed to fetch fuzzy organization users")
			} else if len(u) > 0 {
				a.users = append(a.users, u...)
			}
		}

		if org.ExpandRepo {
			r, err := a.gs.ListOrgRepositories(ctx, org.Type, org.Name,
				listRepoOpt)
			if err != nil {
				log.Error().Err(err).Str("organization", org.Name).
					Str("type", org.Type).
					Msg("failed to fetch organization repositories")
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

	// Blocking request ahead, need more research for avoiding secondary rate
	// limit
	for _, user := range a.users {
		log.Debug().Str("user", user.Name).Msg("processing user")
		r, err := a.gs.ListUserRepositories(ctx, user.Type, user.Name,
			listRepoOpt)
		log.Debug().Str("user", user.Name).Msgf("got repo %d", len(r))
		if err != nil {
			log.Error().Err(err).Str("user", user.Name).Str("type", user.Type).
				Msg("failed to fetch user repositories")
		} else if len(r) > 0 {
			a.repositories = append(a.repositories, r...)
		}
	}
}

func (a *analysis) processRepositories(ctx context.Context) {
	var err error

	sem := semaphore.NewWeighted(int64(a.config.MaxWorker))

	config := *a.config // copy
	sig := a.signature

	findingC := make(chan finding)

	quit := make(chan struct{})
	defer close(quit)

	// collects any findings
	go func() {
		for {
			select {
			case f := <-findingC:
				err := a.db.AddFinding(ctx, f.repository.Name, f.fileName, f.commitHash, f.matches)
				log.Error().Err(err).
					Str("repo", f.repository.Name).
					Str("commit", f.commitHash).
					Str("filename", f.fileName).
					Msg("failed to add finding")
			case <-quit:
				return
			}
		}
	}()

	for _, repo := range a.repositories {
		if err = sem.Acquire(ctx, 1); err != nil {
			log.Error().Err(err).Msg("Failed to acquire semaphore")
			break
		}

		repo := repo // copy
		repo.LatestCommit, err = a.db.GetRepoLatestCommit(ctx, repo.Name)

		// TODO: benchmark this path, currently we only use one goroutine per
		// repository
		go func() {
			defer sem.Release(1)
			processRepository(repo, config.StorageType, config.StoragePath,
				config.IgnoreFiles, sig, findingC)
		}()
	}

	if err = sem.Acquire(ctx, int64(a.config.MaxWorker)); err != nil {
		log.Error().Err(err).Msg("Failed to acquire semaphore")
	}

	quit <- struct{}{}
}

// Runner run the overall analysis pipeline
func (a *analysis) Runner() {
	// should we add timeout?
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// do we need to stop when error occurs?
	a.processOrganizations(ctx)
	a.processUsers(ctx)
	a.processRepositories(ctx)
}
