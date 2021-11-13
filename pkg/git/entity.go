package git

const (
	// GITHUB service type
	GITHUB = "github"
	// GITLAB service type
	GITLAB = "gitlab"
)

// DefaultListRepositoriesOpt is the default option for list repository
var DefaultListRepositoriesOpt = ListRepositoriesOptions{
	WithFork: false,
}

// MaxWorkerKey is the context key to use with golang.org/x/net/context's
// WithValue function to associate an int value with a context.
var MaxWorkerKey = struct{}{}

// ListRepositoriesOptions is the option struct used when fetching
// users/organizations repositories
type ListRepositoriesOptions struct {
	// WithFork decides if forked repo should be included in the list
	WithFork bool
}
