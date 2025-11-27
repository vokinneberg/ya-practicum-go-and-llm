package types

// QueryResponse represents a query response
type QueryResponse struct {
	Answer   string                 `json:"answer"`
	Context  []string               `json:"context,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
