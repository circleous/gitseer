package analysis

import (
	"context"
	"errors"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/jackc/puddle"

	"github.com/circleous/gitseer/pkg/git"
	"github.com/circleous/gitseer/pkg/gitservice"
)

var (
	memoryStorage      = "memory"
	diskStorage        = "disk"
	defaultMaxWorker   = 10
	defaultStorageType = memoryStorage
	defaultWithFork    = false
)

// OrganizationConfig is per organization configuration struct. At least one of ExpandUser or ExpandRepo needs to be set
type OrganizationConfig struct {
	// Type is the type of git service will be used to query
	Type string `toml:"type"`

	// Name is the name of the organization
	Name string `toml:"name"`

	// ExpandUser if set to true, it will add organization users to the analysis process
	ExpandUser bool `toml:"expand_user"`

	// ExpandRepo if set to true, it will add organization repos to the analysis process
	ExpandRepo bool `toml:"expand_repository"`

	// WithFork if set to true, forked repository will be included into the analysis process
	// override the global with_fork option
	// WithFork bool `toml:"with_fork"`
}

// UserConfig is per user configuration struct.
type UserConfig struct {
	// Type is the type of git service will be used to query
	Type string `toml:"type"`

	// Name is the name of the user
	Name string `toml:"name"`

	// WithFork if set to true, forked repository will be included into the analysis process
	// override the global with_fork option
	// WithFork bool `toml:"with_fork"`
}

// Config is the configuration struct for analysis process. It can be created with ParseConfig.
type Config struct {
	// MaxWorker is the max concurrent goroutine for the signature analysis process
	MaxWorker int `toml:"max_worker"`

	// StorageType is the storage type used for cloning the repository
	StorageType string `toml:"storage_type"`
	StoragePath string `toml:"storage_path"`

	// WithFork if set to true, forked repository will be included into the analysis process
	// default to false
	WithFork bool `toml:"with_fork"`

	// GithubToken
	GithubToken string `toml:"github_token"`

	// Organizations
	Organizations []OrganizationConfig `toml:"organization"`

	// Users
	Users []UserConfig `toml:"user"`

	// RepoURL are git clone-able URLs added directly to the analysis process
	//   Example: https://github.com/v8/v8.git. file:///full/path/to/local/repo
	RepoURL []string `toml:"repositories"`
}

type analysis struct {
	p            *puddle.Pool
	config       *Config
	gs           gitservice.Service
	users        []git.User
	repositories []git.Repository
	// glc *gitservice.GitlabService
}

// Service is the main interface for analysis module
type Service interface {
	Runner()
}

// ParseConfig builds a Config from a toml file
func ParseConfig(configPath string) (*Config, error) {
	var config Config

	meta, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return nil, err
	}

	if !meta.IsDefined("max_worker") {
		config.MaxWorker = defaultMaxWorker
	}

	if !meta.IsDefined("storage_type") {
		config.StorageType = defaultStorageType
	}

	if config.StorageType != memoryStorage && config.StorageType != diskStorage {
		return nil, errors.New("invalid storage type choose either \"memory\" or \"disk\"")
	}

	if config.StorageType == diskStorage && !meta.IsDefined("storage_path") {
		return nil, errors.New("disk storage type need storage_path to be defined")
	}

	if fi, err := os.Stat(config.StoragePath); os.IsNotExist(err) || !fi.IsDir() {
		return nil, errors.New("storage_path doesn't exists or not a directory")
	}

	if !meta.IsDefined("with_fork") {
		config.WithFork = defaultWithFork
	}

	return &config, nil
}

// New init analysis
func New(config *Config) (Service, error) {
	var (
		users []git.User
		repos []git.Repository
	)

	parent := context.Background()
	ctx := context.WithValue(parent, git.MaxWorkerKey, config.MaxWorker)

	gs := gitservice.NewGitService(ctx, &gitservice.Options{
		GithubToken: config.GithubToken,
	})

	for _, user := range config.Users {
		users = append(users, git.User{
			Name: user.Name,
			Type: user.Type,
		})
	}

	return &analysis{
		config:       config,
		gs:           gs,
		users:        users,
		repositories: repos,
	}, nil
}
