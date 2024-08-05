package store

import (
	"sync"

	"github.com/OmGuptaIND/recorder"
)

var store *AppStore

type AppStore struct {
	mu         sync.RWMutex
	Recordings map[string]*recorder.Recorder
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
		Recordings: make(map[string]*recorder.Recorder),
	}

	return store
}

// AddRecording adds a recording to the store.
func (s *AppStore) AddRecording(id string, r *recorder.Recorder) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Recordings[id] = r
}

// GetRecording retrieves a recording from the store.
func (s *AppStore) GetRecording(id string) (*recorder.Recorder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.Recordings[id]
	return r, ok
}

// RemoveRecording removes a recording from the store.
func (s *AppStore) RemoveRecording(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Recordings, id)
}

// ListRecordings lists all recordings in the store.
func (s *AppStore) ListRecordings() map[string]recorder.Recorder {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recordings := make(map[string]recorder.Recorder, len(s.Recordings))
	for k, v := range s.Recordings {
		recordings[k] = *v
	}

	return recordings
}
