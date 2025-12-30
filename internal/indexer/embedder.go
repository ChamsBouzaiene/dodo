package indexer

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// NoOpEmbedder is a placeholder embedder that returns zero vectors.
// Useful for testing or when embeddings are not needed.
type NoOpEmbedder struct {
	dimension int
}

// NewNoOpEmbedder creates a no-op embedder.
func NewNoOpEmbedder(dimension int) *NoOpEmbedder {
	return &NoOpEmbedder{dimension: dimension}
}

// Embed returns a zero vector.
func (e *NoOpEmbedder) Embed(ctx context.Context, text string) ([]byte, int, error) {
	vector := make([]float32, e.dimension)
	return encodeVector(vector), e.dimension, nil
}

// EmbedBatch returns zero vectors for all inputs.
func (e *NoOpEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]byte, int, error) {
	vectors := make([][]byte, len(texts))
	for i := range texts {
		vector := make([]float32, e.dimension)
		vectors[i] = encodeVector(vector)
	}
	return vectors, e.dimension, nil
}

// Dimension returns the embedding dimension.
func (e *NoOpEmbedder) Dimension() int {
	return e.dimension
}

// OpenAIEmbedder uses OpenAI's embedding API.
type OpenAIEmbedder struct {
	apiKey    string
	model     string
	dimension int
	client    *http.Client
}

// OpenAI embedding API request/response types
type openAIEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// NewOpenAIEmbedder creates an OpenAI embedder.
// Common models: "text-embedding-3-small" (1536 dims), "text-embedding-3-large" (3072 dims)
func NewOpenAIEmbedder(apiKey, model string, dimension int) *OpenAIEmbedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	if dimension == 0 {
		dimension = 1536 // Default for text-embedding-3-small
	}
	return &OpenAIEmbedder{
		apiKey:    apiKey,
		model:     model,
		dimension: dimension,
		client:    &http.Client{},
	}
}

// Embed generates an embedding for a single text.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]byte, int, error) {
	vectors, dim, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, 0, err
	}
	if len(vectors) == 0 {
		return nil, 0, fmt.Errorf("no embeddings returned")
	}
	return vectors[0], dim, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]byte, int, error) {
	if len(texts) == 0 {
		return [][]byte{}, e.dimension, nil
	}
	
	// Prepare request
	reqBody := openAIEmbeddingRequest{
		Input: texts,
		Model: e.model,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	
	// Send request
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var embResp openAIEmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, 0, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Convert to byte vectors
	vectors := make([][]byte, len(embResp.Data))
	actualDim := 0
	
	for _, data := range embResp.Data {
		if len(data.Embedding) > 0 {
			actualDim = len(data.Embedding)
		}
		vectors[data.Index] = encodeVector(data.Embedding)
	}
	
	// Update dimension if we got actual data
	if actualDim > 0 {
		e.dimension = actualDim
	}
	
	return vectors, e.dimension, nil
}

// Dimension returns the embedding dimension.
func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

// encodeVector encodes a float32 vector to bytes.
// Uses little-endian encoding for compatibility.
func encodeVector(vector []float32) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, vector)
	if err != nil {
		// This should never happen with float32 slices
		panic(fmt.Sprintf("failed to encode vector: %v", err))
	}
	return buf.Bytes()
}

// DecodeVector decodes a byte slice back to a float32 vector.
func DecodeVector(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid vector data length: %d", len(data))
	}
	
	numFloats := len(data) / 4
	vector := make([]float32, numFloats)
	
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &vector)
	if err != nil {
		return nil, fmt.Errorf("failed to decode vector: %w", err)
	}
	
	return vector, nil
}

// NewEmbedderFromEnv creates an embedder based on environment variables.
// Falls back to NoOpEmbedder if no API key is available.
func NewEmbedderFromEnv() Embedder {
	// Try OpenAI first
	apiKey := "" // You'd get this from os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		return NewOpenAIEmbedder(apiKey, "text-embedding-3-small", 1536)
	}
	
	// Fallback to no-op
	return NewNoOpEmbedder(384) // Small dimension for testing
}

