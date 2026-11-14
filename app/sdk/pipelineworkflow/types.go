package workflow

// ProcessSourceInput is the workflow input. It carries file ids, never blobs or storage locations.
type ProcessSourceInput struct {
	JobID   string `json:"job_id"`
	OwnerID string `json:"owner_id"`
	FileID  string `json:"file_id"`
}

// ResolveSourceInput is the input for source resolution.
type ResolveSourceInput struct {
	JobID   string `json:"job_id"`
	OwnerID string `json:"owner_id"`
	FileID  string `json:"file_id"`
}

// ResolveSourceOutput describes the source file the later steps read.
type ResolveSourceOutput struct {
	JobID       string `json:"job_id"`
	OwnerID     string `json:"owner_id"`
	ProjectID   string `json:"project_id"`
	FileID      string `json:"file_id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
}

// ExtractAudioInput is the input for audio extraction.
type ExtractAudioInput struct {
	JobID     string `json:"job_id"`
	OwnerID   string `json:"owner_id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

// ExtractAudioOutput is the audio file the speech step reads.
type ExtractAudioOutput struct {
	JobID   string `json:"job_id"`
	OwnerID string `json:"owner_id"`
	FileID  string `json:"file_id"`
	Name    string `json:"name"`
}

// TranscribeInput is the input for transcription.
type TranscribeInput struct {
	JobID     string `json:"job_id"`
	OwnerID   string `json:"owner_id"`
	ProjectID string `json:"project_id"`
	FileID    string `json:"file_id"`
	Name      string `json:"name"`
}

// TranscribeOutput is the transcript file.
type TranscribeOutput struct {
	JobID  string `json:"job_id"`
	FileID string `json:"file_id"`
}

// ExtractKeyFramesInput is the input for key-frame extraction.
type ExtractKeyFramesInput struct {
	JobID       string `json:"job_id"`
	OwnerID     string `json:"owner_id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
}

// ExtractKeyFramesOutput is the frame file. FileID is empty for audio.
type ExtractKeyFramesOutput struct {
	JobID  string `json:"job_id"`
	FileID string `json:"file_id"`
}

// ComposeExportInput is the input for document composition.
type ComposeExportInput struct {
	JobID            string `json:"job_id"`
	OwnerID          string `json:"owner_id"`
	ProjectID        string `json:"project_id"`
	FileID           string `json:"file_id"`
	Name             string `json:"name"`
	TranscriptFileID string `json:"transcript_file_id"`
	KeyFramesFileID  string `json:"key_frames_file_id"`
}

// ComposeExportOutput is the document file.
type ComposeExportOutput struct {
	JobID  string `json:"job_id"`
	FileID string `json:"file_id"`
}

// RecordRunInput is the ending sentence for a run.
type RecordRunInput struct {
	JobID   string `json:"job_id"`
	Fact    string `json:"fact"`
	Message string `json:"message,omitempty"`
}
