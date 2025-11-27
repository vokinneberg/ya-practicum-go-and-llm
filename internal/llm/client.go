package llm

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Client wraps OpenAI client and provides RAG-specific methods
type Client struct {
	client     *openai.Client
	model      string
	embedModel string
}

// NewClient creates a new LLM client with API key
func NewClient(apiKey, model, embedModel string) *Client {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &Client{
		client:     &client,
		model:      model,
		embedModel: embedModel,
	}
}
