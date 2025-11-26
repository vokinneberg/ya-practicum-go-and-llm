package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

// GenerateAnswer generates an answer using the LLM with context
func (c *Client) GenerateAnswer(ctx context.Context, contextText, question string) (string, error) {
	// Try to load prompts, with fallback to defaults
	systemPrompt := "Ты - помощник, который отвечает на вопросы на основе предоставленного контекста.\nОтвечай точно и по делу, используя только информацию из контекста.\nЕсли в контексте нет информации для ответа, скажи об этом."

	// Try multiple possible paths
	promptPaths := []string{
		"prompts/system_prompt.txt",
		"./prompts/system_prompt.txt",
		"../prompts/system_prompt.txt",
	}
	for _, path := range promptPaths {
		if p, err := loadPrompt(path); err == nil {
			systemPrompt = p
			break
		}
	}

	answerPromptTemplate := "Используй контекст ниже, чтобы ответить на вопрос.\n\nКонтекст:\n{context}\n\nВопрос: {question}\n\nДай точный технический ответ на основе предоставленного контекста."

	answerPaths := []string{
		"prompts/answer_prompt.txt",
		"./prompts/answer_prompt.txt",
		"../prompts/answer_prompt.txt",
	}
	for _, path := range answerPaths {
		if p, err := loadPrompt(path); err == nil {
			answerPromptTemplate = p
			break
		}
	}

	// Replace placeholders
	answerPrompt := strings.ReplaceAll(answerPromptTemplate, "{context}", contextText)
	answerPrompt = strings.ReplaceAll(answerPrompt, "{question}", question)

	// Create chat completion using OpenAI Go client
	systemMsg := openai.SystemMessage(systemPrompt)
	userMsg := openai.UserMessage(answerPrompt)
	res, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(c.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			systemMsg,
			userMsg,
		},
		Temperature: param.Opt[float64]{Value: 0.7},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate completion: %w", err)
	}

	if len(res.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return res.Choices[0].Message.Content, nil
}

// GenerateEmbedding generates an embedding for the given text
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	input := openai.EmbeddingNewParamsInputUnion{
		OfString: param.Opt[string]{Value: text},
	}
	res, err := c.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(c.embedModel),
		Input: input,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(res.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	// Convert []float64 to []float32 for Qdrant
	embedding := make([]float32, len(res.Data[0].Embedding))
	for i, v := range res.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// loadPrompt loads a prompt from a file
func loadPrompt(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
