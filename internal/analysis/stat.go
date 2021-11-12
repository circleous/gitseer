package analysis

import "sync"

type analysisStat struct {
	mu            sync.Mutex
	organizations uint
	users         uint
	repositories  uint
}

func (as *analysisStat) IncreaseOrganization(value uint) {
	as.mu.Lock()
	as.organizations += value
	as.mu.Unlock()
}

func (as *analysisStat) IncreaseUser(value uint) {
	as.mu.Lock()
	as.users += value
	as.mu.Unlock()
}

func (as *analysisStat) IncreaseRepositories(value uint) {
	as.mu.Lock()
	as.repositories += value
	as.mu.Unlock()
}

func (as *analysisStat) GetOrganization() uint {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.organizations
}

func (as *analysisStat) GetUser() uint {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.users
}

func (as *analysisStat) GetRepositories() uint {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.repositories
}
