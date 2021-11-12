package analysis

import (
	"net/url"
	"path"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rs/zerolog/log"

	cgit "github.com/circleous/gitseer/pkg/git"
)

func processRepository(repo cgit.Repository, config Config) {
	var storer storage.Storer
	var wt billy.Filesystem
	var clonedRepository *git.Repository
	var repoPath string

	if config.StorageType == memoryStorage {
		wt = memfs.New()
		storer = memory.NewStorage()
	} else if config.StorageType == diskStorage {
		u, err := url.Parse(repo.URL)
		if err != nil {
			log.Error().Err(err).Str("url", repo.URL).Msg("invalid repository url")
			return
		}

		// FIXME: change u.PATH to repo.Name
		repoPath = path.Join(config.StoragePath, u.Host, u.Path)
		wt = osfs.New(repoPath)
		dot, _ := wt.Chroot(".git")
		storer = filesystem.NewStorage(dot, cache.NewObjectLRUDefault())
	}

	clonedRepository, err := git.Clone(storer, wt, &git.CloneOptions{
		URL: repo.URL,
	})
	if err == git.ErrRepositoryAlreadyExists {
		// if there is already a repository, only chance that the session also using disk storage
		// so we can go ahead open and pull
		clonedRepository, err = git.PlainOpen(repoPath)
		if err != nil {
			log.Error().Err(err).Str("path", repoPath).Msg("failed to open repository")
			return
		}

		worktree, err := clonedRepository.Worktree()
		if err != nil {
			log.Error().Err(err).Str("path", repoPath).Msg("failed to get the .git directory")
			return
		}

		err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			log.Error().Err(err).Str("path", repoPath).Msg("failed to pull")
			return
		}
	} else if err != nil {
		log.Error().Err(err).Str("url", repo.URL).Msg("failed to clone repository")
		return
	}

	commits, err := clonedRepository.CommitObjects()
	if err != nil {
		log.Error().Err(err).Str("url", repo.URL).Msg("failed to get commit from the repository")
		return
	}

	for {
		commit, err := commits.Next()
		if err != nil {
			break
		}

		tree, err := commit.Tree()
		if err != nil {
			log.Error().Err(err).Str("url", repo.URL).Str("commit", commit.Hash.String()).Msg("failed to get tree commit from the repository")
			continue
		}

		files := tree.Files()
		for {
			file, err := files.Next()
			if err != nil {
				break
			}

			log.Debug().Str("filename", file.Name).Str("commit", commit.Hash.String()).Msg("file")
		}
	}
}
