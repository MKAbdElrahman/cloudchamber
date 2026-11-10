package workflow

// ProcessSourceInput is the workflow input. It carries refs/IDs only, never blobs.
type ProcessSourceInput struct {
	JobID     string `json:"job_id"`
	SourceRef string `json:"source_ref"`
}

// ProcessSourceResult is the workflow output. It carries refs/IDs only.
type ProcessSourceResult struct {
	JobID        string `json:"job_id"`
	SourceRef    string `json:"source_ref"`
	AudioRef     string `json:"audio_ref"`
	Transcript   string `json:"transcript_ref"`
	KeyFramesRef string `json:"key_frames_ref"`
	DocumentRef  string `json:"document_ref"`
}

// ResolveSourceInput is the input for source resolution.
type ResolveSourceInput struct {
	JobID     string `json:"job_id"`
	SourceRef string `json:"source_ref"`
}

// ResolveSourceOutput is the output for source resolution.
type ResolveSourceOutput struct {
	JobID         string `json:"job_id"`
	ResolvedRef   string `json:"resolved_ref"`
	SourceType    string `json:"source_type"`
	DurationHints string `json:"duration_hints"`
}

// ExtractAudioInput is the input for audio extraction.
type ExtractAudioInput struct {
	JobID       string `json:"job_id"`
	ResolvedRef string `json:"resolved_ref"`
}

// ExtractAudioOutput is the output for audio extraction.
type ExtractAudioOutput struct {
	JobID    string `json:"job_id"`
	AudioRef string `json:"audio_ref"`
}

// TranscribeInput is the input for transcription.
type TranscribeInput struct {
	JobID    string `json:"job_id"`
	AudioRef string `json:"audio_ref"`
}

// TranscribeOutput is the output for transcription.
type TranscribeOutput struct {
	JobID         string `json:"job_id"`
	TranscriptRef string `json:"transcript_ref"`
}

// ExtractKeyFramesInput is the input for key-frame extraction.
type ExtractKeyFramesInput struct {
	JobID         string `json:"job_id"`
	ResolvedRef   string `json:"resolved_ref"`
	TranscriptRef string `json:"transcript_ref"`
}

// ExtractKeyFramesOutput is the output for key-frame extraction.
type ExtractKeyFramesOutput struct {
	JobID        string `json:"job_id"`
	KeyFramesRef string `json:"key_frames_ref"`
}

// ComposeExportInput is the input for document composition/export.
type ComposeExportInput struct {
	JobID         string `json:"job_id"`
	SourceRef     string `json:"source_ref"`
	TranscriptRef string `json:"transcript_ref"`
	KeyFramesRef  string `json:"key_frames_ref"`
}

// ComposeExportOutput is the output for document composition/export.
type ComposeExportOutput struct {
	JobID       string `json:"job_id"`
	DocumentRef string `json:"document_ref"`
}
