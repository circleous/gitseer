package analysis

import (
	"errors"
	"net/url"
	"path"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rs/zerolog/log"

	cgit "github.com/circleous/gitseer/pkg/git"
	"github.com/circleous/gitseer/pkg/signature"
)

func processFile(file *object.File, commit *object.Commit, repo cgit.Repository, ignoreFiles []string,
	signatures []signature.Base) ([]signature.Match, error) {

	// skip if binary
	if ok, err := file.IsBinary(); err != nil && ok {
		return nil, nil
	}

	filename := file.Name

	// check for ignored files
	imatch := false
	for _, ignoreFile := range ignoreFiles {
		if imatch, _ = filepath.Match(ignoreFile, filename); imatch {
			break
		}
	}

	// if there's a match in ignored pattern, skip
	if imatch {
		return nil, nil
	}

	// get the file content
	content, err := file.Contents()
	if err != nil {
		log.Error().Err(err).Str("url", repo.URL).
			Str("commit", commit.Hash.String()).
			Str("path", filename).
			Msg("failed to get file content")
		return nil, err
	}

	// find any matches with signatures for file name and contents
	matches := signature.ExtractMatch(filename, content, signatures)

	return matches, nil
}

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

	log.Print(repo)

	for {
		commit, err := commits.Next()
		if err != nil {
			break
		}

		// get parent commit, if there isn't any, this could be the first commit,
		// so we don't have to compare anything
		parentCommit, err := commit.Parent(1)
		if err != nil && !errors.Is(err, object.ErrParentNotFound) {
			log.Error().Err(err).Str("url", repo.URL).
				Str("commit", commit.Hash.String()).
				Msg("failed to get parent commit from the repository")
			continue
		}

		tree, err := commit.Tree()
		if err != nil {
			log.Error().Err(err).Str("url", repo.URL).
				Str("commit", commit.Hash.String()).
				Msg("failed to get tree commit from the repository")
			continue
		}

		// log.Print(repo.Name, commit.Hash.String())

		if parentCommit != nil {
			patch, err := parentCommit.Patch(commit)
			if err != nil {
				log.Error().Err(err).Str("url", repo.URL).
					Str("commit", commit.Hash.String()).
					Str("parent", parentCommit.Hash.String()).
					Msg("failed to get patch")
				continue
			}
			filePatches := patch.FilePatches()

			for _, file := range filePatches {
				_, to := file.Files()
				// file could be deleted in this patch
				if to == nil {
					continue
				}

				file, err := tree.File(to.Path())
				if err != nil {
					log.Error().Err(err).Str("url", repo.URL).
						Str("commit", commit.Hash.String()).
						Str("path", to.Path()).
						Msg("failed to get file")
					continue
				}

				matches, err := processFile(file, commit, repo, ignoreFiles, signatures)
				if err != nil {
					log.Error().Err(err).Str("url", repo.URL).
						Str("commit", commit.Hash.String()).
						Str("path", file.Name).
						Msg("failed to process file")
					continue
				}

				// skip if there isn't any match(s)
				if matches != nil || len(matches) == 0 {
					continue
				}

				log.Debug().Str("repo", repo.Name).
					Str("commit", commit.Hash.String()).
					Str("path", file.Name).
					Msgf("found %v", matches)

				findingC <- finding{
					repository: repo,
					commitHash: commit.Hash.String(),
					fileName:   file.Name,
					matches:    matches,
				}
			}
		} else {
			files := tree.Files()
			for {
				file, err := files.Next()
				if err != nil {
					break
				}

				matches, err := processFile(file, commit, repo, ignoreFiles, signatures)
				if err != nil {
					log.Error().Err(err).Str("url", repo.URL).
						Str("commit", commit.Hash.String()).
						Str("path", file.Name).
						Msg("failed to process file")
					continue
				}

				// skip if there isn't any match(s)
				if matches != nil || len(matches) == 0 {
					continue
				}

				findingC <- finding{
					repository: repo,
					commitHash: commit.Hash.String(),
					fileName:   file.Name,
					matches:    matches,
				}
			}
		}
	}
}
