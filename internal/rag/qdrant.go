package rag

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

// QdrantClient wraps Qdrant client and provides RAG-specific methods
type QdrantClient struct {
	client     *qdrant.Client
	collection string
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(host string, port int, collection string) (*QdrantClient, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	qc := &QdrantClient{
		client:     client,
		collection: collection,
	}

	return qc, nil
}

// EnsureCollection ensures the collection exists with the correct configuration
func (qc *QdrantClient) EnsureCollection(ctx context.Context, vectorSize uint64) error {
	// Check if collection exists by trying to get it
	_, err := qc.client.GetCollectionInfo(ctx, qc.collection)
	if err == nil {
		return nil // Collection exists
	}

	// Create collection if it doesn't exist
	err = qc.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: qc.collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// UpsertPoints upserts points (documents) into the collection
func (qc *QdrantClient) UpsertPoints(ctx context.Context, pointsToUpsert []*qdrant.PointStruct) error {
	_, err := qc.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qc.collection,
		Points:         pointsToUpsert,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}
	return nil
}

// Search searches for similar vectors in the collection using Qdrant Query API
func (qc *QdrantClient) Search(ctx context.Context, vector []float32, limit uint64) ([]string, []float32, error) {
	// Use Query API for search
	searchResult, err := qc.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: qc.collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(searchResult) == 0 {
		return []string{}, []float32{}, nil
	}

	texts := make([]string, 0, len(searchResult))
	scores := make([]float32, 0, len(searchResult))

	for _, result := range searchResult {
		// Extract score
		score := float32(result.Score)

		// Extract text from payload
		if result.Payload != nil {
			if textValue, ok := result.Payload["text"]; ok && textValue != nil {
				// Check if it's a string value
				if textValue.GetStringValue() != "" {
					texts = append(texts, textValue.GetStringValue())
					scores = append(scores, score)
				}
			}
		}
	}

	return texts, scores, nil
}
