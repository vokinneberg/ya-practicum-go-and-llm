package rag

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/qdrant/go-client/qdrant"
)

func TestNewPipeline(t *testing.T) {
	tests := []struct {
		name         string
		chunker      TextChunker
		llmClient    LLMClient
		qdrantClient VectorDatabase
		searchLimit  int
		setupMocks   func(*MockVectorDatabase)
		wantErr      bool
		errContains  string
	}{
		{
			name:        "successful creation",
			chunker:     NewChunker(100, 20),
			searchLimit: 3,
			setupMocks: func(m *MockVectorDatabase) {
				m.EXPECT().EnsureCollection(gomock.Any(), uint64(3072)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "collection creation fails",
			chunker:     NewChunker(100, 20),
			searchLimit: 3,
			setupMocks: func(m *MockVectorDatabase) {
				m.EXPECT().EnsureCollection(gomock.Any(), uint64(3072)).Return(errors.New("connection failed"))
			},
			wantErr:     true,
			errContains: "failed to ensure collection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLLM := NewMockLLMClient(ctrl)
			mockQdrant := NewMockVectorDatabase(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(mockQdrant)
			}

			pipeline, err := NewPipeline(tt.chunker, mockLLM, mockQdrant, tt.searchLimit)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPipeline() expected error but got nil")
					return
				}
				if tt.errContains != "" {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("NewPipeline() error = %v, want error containing %q", err, tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("NewPipeline() unexpected error: %v", err)
				return
			}

			if pipeline == nil {
				t.Fatal("NewPipeline() returned nil pipeline")
			}

			if pipeline.searchLimit != tt.searchLimit {
				t.Errorf("NewPipeline() searchLimit = %d, want %d", pipeline.searchLimit, tt.searchLimit)
			}
		})
	}
}

func TestPipeline_Ingest(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		docID       string
		setupMocks  func(*MockTextChunker, *MockLLMClient, *MockVectorDatabase)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful ingestion",
			text:  "This is a test document with multiple words",
			docID: "doc1",
			setupMocks: func(chunker *MockTextChunker, llm *MockLLMClient, db *MockVectorDatabase) {
				chunks := []string{"This is a test", "test document with", "with multiple words"}
				chunker.EXPECT().ChunkText("This is a test document with multiple words").Return(chunks)

				// Create embeddings with correct dimension (3072 for text-embedding-3-large)
				embedding1 := make([]float32, 3072)
				embedding2 := make([]float32, 3072)
				embedding3 := make([]float32, 3072)
				for i := range embedding1 {
					embedding1[i] = float32(i) * 0.001
					embedding2[i] = float32(i) * 0.002
					embedding3[i] = float32(i) * 0.003
				}

				llm.EXPECT().GenerateEmbedding(gomock.Any(), "This is a test").Return(embedding1, nil)
				llm.EXPECT().GenerateEmbedding(gomock.Any(), "test document with").Return(embedding2, nil)
				llm.EXPECT().GenerateEmbedding(gomock.Any(), "with multiple words").Return(embedding3, nil)

				db.EXPECT().UpsertPoints(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, points []*qdrant.PointStruct) error {
						if len(points) != 3 {
							return errors.New("unexpected number of points")
						}
						return nil
					},
				)
			},
			wantErr: false,
		},
		{
			name:  "empty text after chunking",
			text:  "short",
			docID: "doc1",
			setupMocks: func(chunker *MockTextChunker, llm *MockLLMClient, db *MockVectorDatabase) {
				chunker.EXPECT().ChunkText("short").Return([]string{})
			},
			wantErr:     true,
			errContains: "no chunks created",
		},
		{
			name:  "embedding generation fails",
			text:  "test document",
			docID: "doc1",
			setupMocks: func(chunker *MockTextChunker, llm *MockLLMClient, db *MockVectorDatabase) {
				chunker.EXPECT().ChunkText("test document").Return([]string{"test document"})
				llm.EXPECT().GenerateEmbedding(gomock.Any(), "test document").Return(nil, errors.New("API error"))
			},
			wantErr:     true,
			errContains: "failed to generate embedding",
		},
		{
			name:  "upsert fails",
			text:  "test document",
			docID: "doc1",
			setupMocks: func(chunker *MockTextChunker, llm *MockLLMClient, db *MockVectorDatabase) {
				chunker.EXPECT().ChunkText("test document").Return([]string{"test document"})
				embedding := make([]float32, 3072)
				for i := range embedding {
					embedding[i] = float32(i) * 0.001
				}
				llm.EXPECT().GenerateEmbedding(gomock.Any(), "test document").Return(embedding, nil)
				db.EXPECT().UpsertPoints(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
			},
			wantErr:     true,
			errContains: "failed to upsert points",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockChunker := NewMockTextChunker(ctrl)
			mockLLM := NewMockLLMClient(ctrl)
			mockDB := NewMockVectorDatabase(ctrl)

			// Ensure collection exists - must be set up before NewPipeline
			mockDB.EXPECT().EnsureCollection(gomock.Any(), uint64(3072)).Return(nil)

			if tt.setupMocks != nil {
				tt.setupMocks(mockChunker, mockLLM, mockDB)
			}

			pipeline, err := NewPipeline(mockChunker, mockLLM, mockDB, 3)
			if err != nil {
				t.Fatalf("NewPipeline() failed: %v", err)
			}

			err = pipeline.Ingest(context.Background(), tt.text, tt.docID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Ingest() expected error but got nil")
					return
				}
				if tt.errContains != "" {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("Ingest() error = %v, want error containing %q", err, tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Ingest() unexpected error: %v", err)
			}
		})
	}
}

func TestPipeline_Retrieve(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		setupMocks   func(*MockLLMClient, *MockVectorDatabase)
		wantErr      bool
		errContains  string
		wantContains string
	}{
		{
			name:  "successful retrieval",
			query: "test query",
			setupMocks: func(llm *MockLLMClient, db *MockVectorDatabase) {
				queryEmbedding := make([]float32, 3072)
				for i := range queryEmbedding {
					queryEmbedding[i] = float32(i) * 0.001
				}
				llm.EXPECT().GenerateEmbedding(gomock.Any(), "test query").Return(queryEmbedding, nil)

				texts := []string{"Document 1", "Document 2"}
				scores := []float32{0.9, 0.8}
				db.EXPECT().Search(gomock.Any(), queryEmbedding, uint64(3)).Return(texts, scores, nil)
			},
			wantErr:      false,
			wantContains: "Document 1",
		},
		{
			name:  "embedding generation fails",
			query: "test query",
			setupMocks: func(llm *MockLLMClient, db *MockVectorDatabase) {
				llm.EXPECT().GenerateEmbedding(gomock.Any(), "test query").Return(nil, errors.New("API error"))
			},
			wantErr:     true,
			errContains: "failed to generate query embedding",
		},
		{
			name:  "search fails",
			query: "test query",
			setupMocks: func(llm *MockLLMClient, db *MockVectorDatabase) {
				queryEmbedding := make([]float32, 3072)
				for i := range queryEmbedding {
					queryEmbedding[i] = float32(i) * 0.001
				}
				llm.EXPECT().GenerateEmbedding(gomock.Any(), "test query").Return(queryEmbedding, nil)
				db.EXPECT().Search(gomock.Any(), queryEmbedding, uint64(3)).Return(nil, nil, errors.New("search error"))
			},
			wantErr:     true,
			errContains: "failed to search",
		},
		{
			name:  "no results found",
			query: "test query",
			setupMocks: func(llm *MockLLMClient, db *MockVectorDatabase) {
				queryEmbedding := make([]float32, 3072)
				for i := range queryEmbedding {
					queryEmbedding[i] = float32(i) * 0.001
				}
				llm.EXPECT().GenerateEmbedding(gomock.Any(), "test query").Return(queryEmbedding, nil)
				db.EXPECT().Search(gomock.Any(), queryEmbedding, uint64(3)).Return([]string{}, []float32{}, nil)
			},
			wantErr:     true,
			errContains: "no relevant documents found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockChunker := NewMockTextChunker(ctrl)
			mockLLM := NewMockLLMClient(ctrl)
			mockDB := NewMockVectorDatabase(ctrl)

			// Ensure collection exists - must be set up before NewPipeline
			mockDB.EXPECT().EnsureCollection(gomock.Any(), uint64(3072)).Return(nil)

			if tt.setupMocks != nil {
				tt.setupMocks(mockLLM, mockDB)
			}

			pipeline, err := NewPipeline(mockChunker, mockLLM, mockDB, 3)
			if err != nil {
				t.Fatalf("NewPipeline() failed: %v", err)
			}

			result, err := pipeline.Retrieve(context.Background(), tt.query)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Retrieve() expected error but got nil")
					return
				}
				if tt.errContains != "" {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("Retrieve() error = %v, want error containing %q", err, tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Retrieve() unexpected error: %v", err)
				return
			}

			if tt.wantContains != "" {
				if !strings.Contains(result, tt.wantContains) {
					t.Errorf("Retrieve() result = %q, want containing %q", result, tt.wantContains)
				}
			}
		})
	}
}
