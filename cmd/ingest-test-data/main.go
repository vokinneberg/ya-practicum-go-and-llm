package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		slog.Error("Usage: go run scripts/ingest_testdata.go <server-url>")
		os.Exit(1)
	}

	serverURL := os.Args[1]
	testDataDir := "testdata/docs"

	// Read all .txt files from testdata/docs
	files, err := filepath.Glob(filepath.Join(testDataDir, "*.txt"))
	if err != nil {
		slog.Error("Failed to read testdata directory", "error", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		slog.Error("No .txt files found in testdata/docs")
		os.Exit(1)
	}

	// Ingest each file
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			slog.Error("Failed to read file", "file", file, "error", err)
			continue
		}

		// Prepare request
		reqBody := map[string]interface{}{
			"text": string(content),
			"id":   filepath.Base(file),
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			slog.Error("Failed to marshal request", "file", file, "error", err)
			continue
		}
		url := fmt.Sprintf("%s/ingest", serverURL)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			slog.Error("Failed to create request", "file", file, "error", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Failed to ingest file", "file", file, "error", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("Failed to ingest file", "file", file, "status", resp.StatusCode)
			continue
		}

		slog.Info("Successfully ingested file", "file", file)
	}

	slog.Info("Ingestion complete!")
}
