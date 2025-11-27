package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vokinneberg/ya-practicum-go-and-llm/internal/config"
	"github.com/vokinneberg/ya-practicum-go-and-llm/internal/llm"
	"github.com/vokinneberg/ya-practicum-go-and-llm/internal/rag"

	httphandler "github.com/vokinneberg/ya-practicum-go-and-llm/internal/http"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize LLM client
	llmClient := llm.NewClient(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIEmbedModel)
	slog.Info("Initialized OpenAI client")

	// Initialize Qdrant client
	qdrantClient, err := rag.NewQdrantClient(cfg.QdrantHost, cfg.QdrantPort, cfg.QdrantCollection)
	if err != nil {
		slog.Error("Failed to create Qdrant client", "error", err)
		os.Exit(1)
	}
	slog.Info("Initialized Qdrant client")

	// Initialize chunker
	chunker := rag.NewChunker(cfg.ChunkSize, cfg.ChunkOverlap)
	slog.Info("Initialized chunker", "size", cfg.ChunkSize, "overlap", cfg.ChunkOverlap)

	// Initialize RAG pipeline
	pipeline, err := rag.NewPipeline(chunker, llmClient, qdrantClient, cfg.SearchLimit)
	if err != nil {
		slog.Error("Failed to create RAG pipeline", "error", err)
		os.Exit(1)
	}
	slog.Info("Initialized RAG pipeline")

	// Initialize HTTP handlers
	handler := httphandler.NewHandlers(pipeline, llmClient)

	// Create router
	r := httphandler.NewRouter(handler)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Server running", "port", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited")
}
