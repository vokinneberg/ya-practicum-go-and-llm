package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	ServerPort string

	// OpenAI configuration
	OpenAIAPIKey     string
	OpenAIModel      string
	OpenAIEmbedModel string

	// Qdrant configuration
	QdrantHost       string
	QdrantPort       int
	QdrantCollection string

	// RAG configuration
	ChunkSize    int
	ChunkOverlap int
	SearchLimit  int
}

// LoadConfig loads configuration from environment variables and command-line flags
// Flags take precedence over environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Define flags
	serverPort := flag.String("server-port", getEnv("SERVER_PORT", "8080"), "Server port")
	openAIKey := flag.String("openai-key", getEnv("OPENAI_API_KEY", ""), "OpenAI API key")
	openAIModel := flag.String("openai-model", getEnv("OPENAI_MODEL", "gpt-4.1-mini"), "OpenAI model for chat completions")
	openAIEmbedModel := flag.String("openai-embed-model", getEnv("OPENAI_EMBED_MODEL", "text-embedding-3-large"), "OpenAI model for embeddings")
	qdrantHost := flag.String("qdrant-host", getEnv("QDRANT_HOST", "localhost"), "Qdrant host")
	qdrantPort := flag.Int("qdrant-port", getEnvAsInt("QDRANT_PORT", 6334), "Qdrant gRPC port (default: 6334)")
	qdrantCollection := flag.String("qdrant-collection", getEnv("QDRANT_COLLECTION", "docs"), "Qdrant collection name")
	chunkSize := flag.Int("chunk-size", getEnvAsInt("CHUNK_SIZE", 1000), "Text chunk size")
	chunkOverlap := flag.Int("chunk-overlap", getEnvAsInt("CHUNK_OVERLAP", 200), "Text chunk overlap")
	searchLimit := flag.Int("search-limit", getEnvAsInt("SEARCH_LIMIT", 3), "Number of search results to return")

	flag.Parse()

	// Set config values
	cfg.ServerPort = *serverPort
	cfg.OpenAIAPIKey = *openAIKey
	cfg.OpenAIModel = *openAIModel
	cfg.OpenAIEmbedModel = *openAIEmbedModel
	cfg.QdrantHost = *qdrantHost
	cfg.QdrantPort = *qdrantPort
	cfg.QdrantCollection = *qdrantCollection
	cfg.ChunkSize = *chunkSize
	cfg.ChunkOverlap = *chunkOverlap
	cfg.SearchLimit = *searchLimit

	// Validate required fields
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required (set via environment variable or -openai-key flag)")
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
