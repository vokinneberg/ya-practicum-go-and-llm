package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/vokinneberg/ya-practicum-go-and-llm/internal/types"
)

func TestHandler_QueryHandler(t *testing.T) {
	tests := []struct {
		name         string
		requestBody  interface{}
		setupMocks   func(*MockRAGPipeline, *MockLLMClient)
		wantStatus   int
		wantContains string
	}{
		{
			name: "successful query",
			requestBody: QueryReq{
				Query: "What is Kubernetes?",
			},
			setupMocks: func(pipeline *MockRAGPipeline, llm *MockLLMClient) {
				pipeline.EXPECT().
					Retrieve(gomock.Any(), "What is Kubernetes?").
					Return("Kubernetes is a container orchestration system", nil)
				llm.EXPECT().
					GenerateAnswer(gomock.Any(), "Kubernetes is a container orchestration system", "What is Kubernetes?").
					Return("Kubernetes is a container orchestration platform", nil)
			},
			wantStatus:   http.StatusOK,
			wantContains: "Kubernetes is a container orchestration platform",
		},
		{
			name:        "invalid JSON",
			requestBody: "invalid json",
			setupMocks:  func(*MockRAGPipeline, *MockLLMClient) {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name: "empty query",
			requestBody: QueryReq{
				Query: "",
			},
			setupMocks: func(*MockRAGPipeline, *MockLLMClient) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "retrieve fails",
			requestBody: QueryReq{
				Query: "test query",
			},
			setupMocks: func(pipeline *MockRAGPipeline, llm *MockLLMClient) {
				pipeline.EXPECT().
					Retrieve(gomock.Any(), "test query").
					Return("", errors.New("retrieve error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "LLM generation fails",
			requestBody: QueryReq{
				Query: "test query",
			},
			setupMocks: func(pipeline *MockRAGPipeline, llm *MockLLMClient) {
				pipeline.EXPECT().
					Retrieve(gomock.Any(), "test query").
					Return("context text", nil)
				llm.EXPECT().
					GenerateAnswer(gomock.Any(), "context text", "test query").
					Return("", errors.New("LLM error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPipeline := NewMockRAGPipeline(ctrl)
			mockLLM := NewMockLLMClient(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockPipeline, mockLLM)
			}

			handler := NewHandlers(mockPipeline, mockLLM)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.QueryHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("QueryHandler() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantContains != "" {
				if !bytes.Contains(w.Body.Bytes(), []byte(tt.wantContains)) {
					t.Errorf("QueryHandler() body = %s, want containing %q", w.Body.String(), tt.wantContains)
				}
			}
		})
	}
}

func TestHandler_IngestHandler(t *testing.T) {
	tests := []struct {
		name        string
		requestBody interface{}
		setupMocks  func(*MockRAGPipeline)
		wantStatus  int
	}{
		{
			name: "successful ingestion",
			requestBody: IngestReq{
				Text: "This is a test document",
				ID:   "doc1",
			},
			setupMocks: func(pipeline *MockRAGPipeline) {
				pipeline.EXPECT().
					Ingest(gomock.Any(), "This is a test document", "doc1").
					Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid JSON",
			requestBody: "invalid json",
			setupMocks:  func(*MockRAGPipeline) {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name: "empty text",
			requestBody: IngestReq{
				Text: "",
				ID:   "doc1",
			},
			setupMocks: func(*MockRAGPipeline) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "ingestion fails",
			requestBody: IngestReq{
				Text: "test document",
				ID:   "doc1",
			},
			setupMocks: func(pipeline *MockRAGPipeline) {
				pipeline.EXPECT().
					Ingest(gomock.Any(), "test document", "doc1").
					Return(errors.New("ingestion error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "ingestion without ID",
			requestBody: IngestReq{
				Text: "test document",
				ID:   "",
			},
			setupMocks: func(pipeline *MockRAGPipeline) {
				pipeline.EXPECT().
					Ingest(gomock.Any(), "test document", "").
					Return(nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPipeline := NewMockRAGPipeline(ctrl)
			mockLLM := NewMockLLMClient(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockPipeline)
			}

			handler := NewHandlers(mockPipeline, mockLLM)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.IngestHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("IngestHandler() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var response map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("IngestHandler() invalid JSON response: %v", err)
				}
				if response["status"] != "success" {
					t.Errorf("IngestHandler() status = %q, want %q", response["status"], "success")
				}
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		err        error
		wantStatus int
		wantError  string
	}{
		{
			name:       "error with message",
			status:     http.StatusBadRequest,
			message:    "Invalid request",
			err:        errors.New("validation failed"),
			wantStatus: http.StatusBadRequest,
			wantError:  "Bad Request",
		},
		{
			name:       "error without message",
			status:     http.StatusInternalServerError,
			message:    "Server error",
			err:        nil,
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			errorResponse(w, tt.status, tt.message, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("errorResponse() status = %d, want %d", w.Code, tt.wantStatus)
			}

			var response types.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("errorResponse() invalid JSON: %v", err)
			}

			if response.Error != tt.wantError {
				t.Errorf("errorResponse() Error = %q, want %q", response.Error, tt.wantError)
			}

			if tt.message != "" {
				if !strings.Contains(response.Message, tt.message) {
					t.Errorf("errorResponse() Message = %q, want containing %q", response.Message, tt.message)
				}
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HealthHandler() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("HealthHandler() invalid JSON: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("HealthHandler() status = %q, want %q", response["status"], "ok")
	}
}


