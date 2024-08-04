package api

import "github.com/OmGuptaIND/recorder"

type StartRecordingRequest struct {
	Url string `json:"url"`
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
