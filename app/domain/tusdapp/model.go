package tusdapp

import "encoding/json"

// hookRequest is the tusd HTTP hook request payload.
// See https://tus.github.io/tusd/advanced-topics/hooks/
type hookRequest struct {
	Type  string    `json:"Type"`
	Event hookEvent `json:"Event"`
}

type hookEvent struct {
	Upload      hookUpload      `json:"Upload"`
	HTTPRequest hookHTTPRequest `json:"HTTPRequest"`
}

type hookUpload struct {
	ID       string            `json:"ID"`
	Size     int64             `json:"Size"`
	Offset   int64             `json:"Offset"`
	MetaData map[string]string `json:"MetaData"`
	Storage  hookStorage       `json:"Storage"`
}

type hookStorage struct {
	Type   string `json:"Type"`
	Bucket string `json:"Bucket"`
	Key    string `json:"Key"`
}

type hookHTTPRequest struct {
	Method     string              `json:"Method"`
	URI        string              `json:"URI"`
	RemoteAddr string              `json:"RemoteAddr"`
	Header     map[string][]string `json:"Header"`
}

// hookResponse is the tusd HTTP hook response payload.
type hookResponse struct {
	HTTPResponse      *hookHTTPResponse `json:"HTTPResponse,omitempty"`
	RejectUpload      bool              `json:"RejectUpload,omitempty"`
	RejectTermination bool              `json:"RejectTermination,omitempty"`
	ChangeFileInfo    *hookFileInfo     `json:"ChangeFileInfo,omitempty"`
}

type hookHTTPResponse struct {
	StatusCode int               `json:"StatusCode"`
	Body       string            `json:"Body"`
	Header     map[string]string `json:"Header,omitempty"`
}

type hookFileInfo struct {
	ID       string            `json:"ID,omitempty"`
	MetaData map[string]string `json:"MetaData,omitempty"`
}

// Encode implements the web.Encoder interface.
func (h hookResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(h)
	return data, "application/json", err
}
