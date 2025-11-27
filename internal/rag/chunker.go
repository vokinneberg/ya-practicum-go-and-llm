package rag

import (
	"os"
	"strings"
)

// Chunker handles text chunking with overlap support
type Chunker struct {
	chunkSize    int
	chunkOverlap int
}

// NewChunker creates a new chunker with specified size and overlap
func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// ChunkText splits text into chunks with overlap
func (c *Chunker) ChunkText(text string) []string {
	if text == "" {
		return []string{}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	chunks := []string{}
	currentChunk := []string{}
	currentSize := 0

	for i, word := range words {
		wordSize := len(word) + 1 // +1 for space

		// If adding this word would exceed chunk size, save current chunk
		if currentSize+wordSize > c.chunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, strings.Join(currentChunk, " "))

			// Handle overlap: keep last N words for overlap
			if c.chunkOverlap > 0 && len(currentChunk) > 0 {
				overlapWords := c.getOverlapWords(currentChunk)
				currentChunk = overlapWords
				currentSize = c.calculateSize(overlapWords)
			} else {
				currentChunk = []string{}
				currentSize = 0
			}
		}

		currentChunk = append(currentChunk, word)
		currentSize += wordSize

		// If this is the last word, add the chunk
		if i == len(words)-1 {
			chunks = append(chunks, strings.Join(currentChunk, " "))
		}
	}

	return chunks
}

// getOverlapWords returns the last N words for overlap
func (c *Chunker) getOverlapWords(words []string) []string {
	if c.chunkOverlap <= 0 {
		return []string{}
	}

	overlapCount := c.chunkOverlap
	if overlapCount > len(words) {
		overlapCount = len(words)
	}

	return words[len(words)-overlapCount:]
}

// calculateSize calculates the total size of words including spaces
func (c *Chunker) calculateSize(words []string) int {
	size := 0
	for _, word := range words {
		size += len(word) + 1 // +1 for space
	}
	if size > 0 {
		size-- // Remove last space
	}
	return size
}

// LoadFile loads a file and returns its content
func LoadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
