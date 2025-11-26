package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vokinneberg/ya-practicum-go-and-llm/internal/types"
)

//go:generate mockgen -source=handlers.go -destination=mock_llmclient.go -package=http LLMClient

// LLMClient defines the interface for LLM answer generation
type LLMClient interface {
	GenerateAnswer(ctx context.Context, contextText, question string) (string, error)
}

//go:generate mockgen -source=handlers.go -destination=mock_ragpipeline.go -package=http RAGPipeline

// RAGPipeline defines the interface for RAG pipeline operations
type RAGPipeline interface {
	Retrieve(ctx context.Context, query string) (string, error)
	Ingest(ctx context.Context, text string, docID string) error
}

type QueryReq struct {
	Query string `json:"query"`
}

type IngestReq struct {
	Text string `json:"text"`
	ID   string `json:"id,omitempty"`
}

type Handler struct {
	ragPipeline RAGPipeline
	llmClient   LLMClient
}

// InitHandlers initializes handlers with dependencies
func NewHandlers(ragPipeline RAGPipeline, llmClient LLMClient) *Handler {
	return &Handler{
		ragPipeline: ragPipeline,
		llmClient:   llmClient,
	}
}

func (h *Handler) QueryHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req QueryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Query == "" {
		errorResponse(w, http.StatusBadRequest, "Query is required", nil)
		return
	}

	ctx := r.Context()

	// RAG pipeline - retrieve relevant context
	contextText, err := h.ragPipeline.Retrieve(ctx, req.Query)
	if err != nil {
		slog.Error("Error retrieving context", "error", err, "query", req.Query)
		errorResponse(w, http.StatusInternalServerError, "Failed to retrieve context", err)
		return
	}

	// LLM generation
	answer, err := h.llmClient.GenerateAnswer(ctx, contextText, req.Query)
	if err != nil {
		slog.Error("Error generating answer", "error", err, "query", req.Query)
		errorResponse(w, http.StatusInternalServerError, "Failed to generate answer", err)
		return
	}

	response := types.QueryResponse{
		Answer: answer,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

func (h *Handler) IngestHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req IngestReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Text == "" {
		errorResponse(w, http.StatusBadRequest, "Text is required", nil)
		return
	}

	ctx := r.Context()

	// Ingest document into RAG pipeline
	if err := h.ragPipeline.Ingest(ctx, req.Text, req.ID); err != nil {
		slog.Error("Error ingesting document", "error", err, "doc_id", req.ID)
		errorResponse(w, http.StatusInternalServerError, "Failed to ingest document", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

func errorResponse(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	if err := json.NewEncoder(w).Encode(types.ErrorResponse{
		Error:   http.StatusText(status),
		Message: errorMsg,
	}); err != nil {
		slog.Error("Error encoding error response", "error", err, "status", status)
	}
}
