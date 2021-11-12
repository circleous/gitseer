package git

// Repository is the struct containing the repo data from user/org
type Repository struct {
	// Name repository name in user/example-git-repo format
	Name string
	// URL git clone-able repo URL
	URL string
	// LatestCommit latest commit hash of the repo
	LatestCommit string
}
