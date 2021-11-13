package analysis

import (
	"net/url"
	"path"
	"path/filepath"

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
	"github.com/circleous/gitseer/pkg/signature"
)

func processRepository(repo cgit.Repository, storageType, storagePath string,
	ignoreFiles []string, signatures []signature.Base, findingC chan finding) {
	var storer storage.Storer
	var wt billy.Filesystem
	var clonedRepository *git.Repository
	var repoPath string

	if storageType == memoryStorage {
		wt = memfs.New()
		storer = memory.NewStorage()
	} else if storageType == diskStorage {
		u, err := url.Parse(repo.URL)
		if err != nil {
			log.Error().Err(err).Str("url", repo.URL).
				Msg("invalid repository url")
			return
		}

		repoPath = path.Join(storagePath, u.Host, repo.Name)
		wt = osfs.New(repoPath)
		dot, _ := wt.Chroot(".git")
		storer = filesystem.NewStorage(dot, cache.NewObjectLRUDefault())
	}

	clonedRepository, err := git.Clone(storer, wt, &git.CloneOptions{
		URL: repo.URL,
	})
	// if there is already a repository, only chance that the session also using
	// disk storage so we can go ahead open and pull
	if err == git.ErrRepositoryAlreadyExists {
		// open local repo, there shouldn't be any error since we know that it's
		// already exists
		clonedRepository, err = git.PlainOpen(repoPath)
		if err != nil {
			log.Error().Err(err).Str("path", repoPath).
				Msg("failed to open repository")
			return
		}

		// open the worktree
		worktree, err := clonedRepository.Worktree()
		if err != nil {
			log.Error().Err(err).Str("path", repoPath).
				Msg("failed to get the .git directory")
			return
		}

		// pull from remote "origin"
		err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			log.Error().Err(err).Str("path", repoPath).Msg("failed to pull")
			return
		}
	} else if err != nil {
		log.Error().Err(err).Str("url", repo.URL).
			Msg("failed to clone repository")
		return
	}

	// iterate through commit objects
	commits, err := clonedRepository.CommitObjects()
	if err != nil {
		log.Error().Err(err).Str("url", repo.URL).
			Msg("failed to get commit from the repository")
		return
	}

	for {
		commit, err := commits.Next()
		if err != nil {
			break
		}

		// for every commmit, iterate throught it's file tree
		// TODO: only process changed files, not the whole tree
		tree, err := commit.Tree()
		if err != nil {
			log.Error().Err(err).Str("url", repo.URL).
				Str("commit", commit.Hash.String()).
				Msg("failed to get tree commit from the repository")
			continue
		}

		files := tree.Files()
		for {
			file, err := files.Next()
			if err != nil {
				break
			}

			filename := file.Name

			// check for ignored files
			imatch := false
			for _, ignoreFile := range ignoreFiles {
				if imatch, _ = filepath.Match(ignoreFile, filename); imatch {
					break
				}
			}
			// if there's a match, skip it
			if imatch {
				continue
			}

			// get the file content
			content, err := file.Contents()

			// find any lmatches with signatures for file name and contents
			lmatches := signature.ExtractMatch(filename, content, signatures)

			// check if there's any match, if not then skip to next file
			if lmatches == nil && len(lmatches) == 0 {
				continue
			}

			findingC <- finding{
				repository: repo,
				commitHash: commit.Hash.String(),
				matches:    lmatches,
			}
		}
	}
}
