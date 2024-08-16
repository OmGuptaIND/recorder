package store

import (
	"sync"

	"github.com/OmGuptaIND/pipeline"
)

var store *AppStore

type AppStore struct {
	mu        sync.RWMutex
	Pipelines map[string]*pipeline.Pipeline
}

// GetStore retrieves the store.
func GetStore() *AppStore {
	return store
}

// NewStore creates a new store.
func NewStore() *AppStore {
	if store != nil {
		return store
	}

	store = &AppStore{
		Pipelines: make(map[string]*pipeline.Pipeline),
	}

	return store
}

// AddPipeLine adds a recording to the store.
func (s *AppStore) AddPipeLine(id string, p *pipeline.Pipeline) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Pipelines[id] = p
}

// GetPipeline retrieves a recording from the store.
func (s *AppStore) GetPipeline(id string) (*pipeline.Pipeline, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.Pipelines[id]
	return r, ok
}

// RemovePipeline removes a recording from the store.
func (s *AppStore) RemovePipeline(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Pipelines, id)
}

// ListPipelines lists all recordings in the store.
func (s *AppStore) ListPipelines() map[string]*pipeline.Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pipelines := make(map[string]*pipeline.Pipeline, len(s.Pipelines))
	for k, v := range s.Pipelines {
		pipelines[k] = v
	}

	return pipelines
}
