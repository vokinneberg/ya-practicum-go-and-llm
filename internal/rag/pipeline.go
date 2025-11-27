package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

//go:generate mockgen -source=pipeline.go -destination=mock_llmclient.go -package=rag LLMClient

// LLMClient defines the interface for LLM operations
type LLMClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateAnswer(ctx context.Context, contextText, question string) (string, error)
}

//go:generate mockgen -source=pipeline.go -destination=mock_textchunker.go -package=rag TextChunker

// TextChunker defines the interface for text chunking operations
type TextChunker interface {
	ChunkText(text string) []string
}

//go:generate mockgen -source=pipeline.go -destination=mock_vectordatabase.go -package=rag VectorDatabase

// VectorDatabase defines the interface for vector database operations
type VectorDatabase interface {
	EnsureCollection(ctx context.Context, vectorSize uint64) error
	UpsertPoints(ctx context.Context, pointsToUpsert []*qdrant.PointStruct) error
	Search(ctx context.Context, queryEmbedding []float32, limit uint64) ([]string, []float32, error)
}

// Pipeline orchestrates the RAG pipeline
type Pipeline struct {
	chunker      TextChunker
	llmClient    LLMClient
	qdrantClient VectorDatabase
	searchLimit  int
}

// NewPipeline creates a new RAG pipeline
func NewPipeline(chunker TextChunker, llmClient LLMClient, qdrantClient VectorDatabase, searchLimit int) (*Pipeline, error) {
	// Ensure collection exists with correct vector size
	// text-embedding-3-large produces 3072-dimensional vectors
	vectorSize := uint64(3072)
	ctx := context.Background()
	if err := qdrantClient.EnsureCollection(ctx, vectorSize); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return &Pipeline{
		chunker:      chunker,
		llmClient:    llmClient,
		qdrantClient: qdrantClient,
		searchLimit:  searchLimit,
	}, nil
}

// Ingest processes and stores a document in the vector database
func (p *Pipeline) Ingest(ctx context.Context, text string, docID string) error {
	// Chunk the text
	chunks := p.chunker.ChunkText(text)

	if len(chunks) == 0 {
		return fmt.Errorf("no chunks created from text")
	}

	// Generate embeddings for each chunk and prepare points
	pointsToUpsert := make([]*qdrant.PointStruct, 0, len(chunks))

	for i, chunk := range chunks {
		// Generate embedding
		embedding, err := p.llmClient.GenerateEmbedding(ctx, chunk)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}

		// Create point ID (use docID + chunk index if docID provided, otherwise use timestamp)
		var pointID uint64
		if docID != "" {
			pointID = uint64(i) // Simple ID based on chunk index
		} else {
			pointID = uint64(time.Now().UnixNano()) + uint64(i)
		}

		// Create point with payload using Qdrant helper functions
		point := &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(pointID),
			Vectors: qdrant.NewVectors(embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"text":        chunk,
				"doc_id":      docID,
				"chunk_index": int64(i),
			}),
		}

		pointsToUpsert = append(pointsToUpsert, point)
	}

	// Upsert points to Qdrant
	if err := p.qdrantClient.UpsertPoints(ctx, pointsToUpsert); err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// Retrieve searches for relevant context based on a query
func (p *Pipeline) Retrieve(ctx context.Context, query string) (string, error) {
	// Generate embedding for the query
	queryEmbedding, err := p.llmClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for similar documents
	texts, scores, err := p.qdrantClient.Search(ctx, queryEmbedding, uint64(p.searchLimit))
	if err != nil {
		return "", fmt.Errorf("failed to search: %w", err)
	}

	if len(texts) == 0 {
		return "", fmt.Errorf("no relevant documents found")
	}

	// Combine retrieved texts into context
	var contextBuilder strings.Builder
	for i, text := range texts {
		contextBuilder.WriteString(fmt.Sprintf("[Document %d, Score: %.4f]\n%s\n\n", i+1, scores[i], text))
	}

	return strings.TrimSpace(contextBuilder.String()), nil
}
