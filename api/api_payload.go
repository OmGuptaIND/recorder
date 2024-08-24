package api

import "github.com/OmGuptaIND/recorder"

type ChunkRequest struct {
	Duration string `json:"duration"`
}

type StartRecordingRequest struct {
	Url       string       `json:"url"`
	StreamUrl string       `json:"stream_url"`
	Chunking  ChunkRequest `json:"chunking,omitempty"`
}

type StartRecordingResponse struct {
	Status string `json:"status"`
	Id     string `json:"id"`
}

type StopRecordingRequest struct {
	Id string `json:"id"`
}

type StopRecordingResponse struct {
	Status string `json:"status"`
}

type ListRecordingsResponse struct {
	Recordings map[string]recorder.Recorder `json:"recordings"`
}
