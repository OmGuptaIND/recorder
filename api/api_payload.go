package api

import "github.com/OmGuptaIND/recorder"

type ChunkRequest struct {
	Duration string `json:"duration"`
}

type StartRecordingRequest struct {
	RecordUrl string `json:"record_url"`
	StreamUrl string `json:"stream_url"`
}

type StartRecordingResponse struct {
	Status string `json:"status"`
	Id     string `json:"id"`
}

type StopRecordingRequest struct {
	Id string `json:"id"`
}

type StopRecordingResponse struct {
	Id           string `json:"id"`
	Status       string `json:"status"`
	RecordingUrl string `json:"recording_url,omitempty"`
}

type ListRecordingsResponse struct {
	Recordings map[string]recorder.Recorder `json:"recordings"`
}
