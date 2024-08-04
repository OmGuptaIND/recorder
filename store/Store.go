package store

import (
	"sync"

	"github.com/OmGuptaIND/recorder"
)

// Store represents a store for recordings.
type Store struct {
	mu         sync.RWMutex
	Recordings map[string]*recorder.Recorder
}

// NewStore creates a new store.
func NewStore() *Store {
	return &Store{
		Recordings: make(map[string]*recorder.Recorder),
	}
}

// AddRecording adds a recording to the store.
func (s *Store) AddRecording(id string, r *recorder.Recorder) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Recordings[id] = r
}

// GetRecording retrieves a recording from the store.
func (s *Store) GetRecording(id string) (*recorder.Recorder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.Recordings[id]
	return r, ok
}

// RemoveRecording removes a recording from the store.
func (s *Store) RemoveRecording(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Recordings, id)
}

// ListRecordings lists all recordings in the store.
func (s *Store) ListRecordings() map[string]recorder.Recorder {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recordings := make(map[string]recorder.Recorder, len(s.Recordings))
	for k, v := range s.Recordings {
		recordings[k] = *v
	}

	return recordings
}
