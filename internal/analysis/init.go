package analysis

import (
	"context"
	"errors"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/circleous/gitseer/internal/database"
	"github.com/circleous/gitseer/pkg/git"
	"github.com/circleous/gitseer/pkg/gitservice"
	"github.com/circleous/gitseer/pkg/signature"
)

var (
	memoryStorage      = "memory"
	diskStorage        = "disk"
	defaultMaxWorker   = 10
	defaultStorageType = memoryStorage
	defaultWithFork    = false
)

// OrganizationConfig is per organization configuration struct. At least one of
// ExpandUser or ExpandRepo needs to be set
type OrganizationConfig struct {
	// Type is the type of git service will be used to query
	Type string `toml:"type"`

	// Name is the name of the organization
	Name string `toml:"name"`

	// ExpandUser if set to true, it will add organization users to analysis
	ExpandUser bool `toml:"expand_user"`

	// ExpandRepo if set to true, it will add organization repos to analysis
	ExpandRepo bool `toml:"expand_repository"`

	// WithFork if set to true, forked repository will be included into analysis
	// override the global with_fork option
	// WithFork bool `toml:"with_fork"`
}

// UserConfig is per user configuration struct.
type UserConfig struct {
	// Type is the type of git service will be used to query
	Type string `toml:"type"`

	// Name is the name of the user
	Name string `toml:"name"`

	// WithFork if set to true, forked repository will be included into analysis
	// override the global with_fork option
	// WithFork bool `toml:"with_fork"`
}

// Config is the configuration struct for analysis process. It can be created
// with ParseConfig.
type Config struct {
	// SignaturePath is the path to signature file
	SignaturePath string `toml:"signature_path"`

	// MaxWorker is the max concurrent goroutine for the analysis process
	MaxWorker int `toml:"max_worker"`

	// StorageType is the storage type used for cloning the repository
	StorageType string `toml:"storage_type"`
	StoragePath string `toml:"storage_path"`

	// WithFork if set to true, forked repository will be included into the
	// analysis process (default false)
	WithFork bool `toml:"with_fork"`

	IgnoreFiles []string `toml:"ignore_files"`

	DatabaseURI string `toml:"database"`

	// GithubToken
	GithubToken string `toml:"github_token"`

	// Organizations
	Organizations []OrganizationConfig `toml:"organization"`

	// Users
	Users []UserConfig `toml:"user"`

	// RepoURL are git clone-able URLs added directly to the analysis process
	//   Example: https://github.com/v8/v8.git, file:///full/path/to/local/repo
	RepoURL []string `toml:"repositories"`
}

type analysis struct {
	config       *Config
	gs           gitservice.Service
	db           database.Service
	users        []git.User
	repositories []git.Repository
	signature    []signature.Base
	finds        []finding
}

type finding struct {
	repository git.Repository
	commitHash string
	fileName   string
	matches    []signature.Match
}

// Service is the main interface for analysis module
type Service interface {
	Runner()
	Close()
}

// ParseConfig builds a Config from a toml file
func ParseConfig(configPath string) (*Config, error) {
	var config Config

	meta, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return nil, err
	}

	if !meta.IsDefined("database") {
		return nil, errors.New("database is not defined")
	}

	if !meta.IsDefined("signature_path") {
		return nil, errors.New("signature_path is not defined")
	}

	if _, err := os.Stat(config.SignaturePath); os.IsNotExist(err) {
		return nil, err
	}

	if config.StorageType != memoryStorage &&
		config.StorageType != diskStorage {
		return nil, errors.New("invalid storage type")
	}

	if config.StorageType == diskStorage && !meta.IsDefined("storage_path") {
		return nil, errors.New("storage_path is not defined")
	}

	if f, err := os.Stat(config.StoragePath); os.IsNotExist(err) || !f.IsDir() {
		return nil, errors.New("storage_path doesn't exists or not a directory")
	}

	if !meta.IsDefined("max_worker") {
		config.MaxWorker = defaultMaxWorker
	}

	if !meta.IsDefined("storage_type") {
		config.StorageType = defaultStorageType
	}

	if !meta.IsDefined("with_fork") {
		config.WithFork = defaultWithFork
	}

	return &config, nil
}

// New init analysis
func New(config *Config, sig []signature.Base) (Service, error) {
	var (
		users []git.User
		repos []git.Repository
		finds []finding
	)

	db, err := database.NewDatabase(config.DatabaseURI)
	if err != nil {
		return nil, err
	}

	if err = db.Initialize(); err != nil {
		return nil, err
	}

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
		db:           db,
		config:       config,
		gs:           gs,
		users:        users,
		repositories: repos,
		signature:    sig,
		finds:        finds,
	}, nil
}

func (a *analysis) Close() {
	a.db.Close()
}
