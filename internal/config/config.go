package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr          string
	OllamaBaseURL     string
	ChatModel         string
	EmbeddingModel    string
	DataDir           string
	StoreDir          string
	ManifestPath      string
	WorkspaceDir      string
	IndexName         string
	ChunkBytes        int
	ChunkOverlap      int
	SearchHitLimit    int
	ContextChunkLimit int
	AutoIndexOnStart  bool
	RequestTimeout    time.Duration
}

func Load() Config {
	dataDir := envString("LOCALAGENT_DATA_DIR", "/data")
	storeDir := filepath.Join(dataDir, "pebble")

	return Config{
		HTTPAddr:          envString("LOCALAGENT_HTTP_ADDR", ":8080"),
		OllamaBaseURL:     envString("LOCALAGENT_OLLAMA_URL", "http://ollama:11434"),
		ChatModel:         envString("LOCALAGENT_CHAT_MODEL", "qwen2.5-coder:7b-instruct"),
		EmbeddingModel:    envString("LOCALAGENT_EMBEDDING_MODEL", "nomic-embed-text"),
		DataDir:           dataDir,
		StoreDir:          storeDir,
		ManifestPath:      filepath.Join(dataDir, "index-manifest.json"),
		WorkspaceDir:      envString("LOCALAGENT_WORKSPACE_DIR", "/workspace"),
		IndexName:         envString("LOCALAGENT_INDEX_NAME", "workspace"),
		ChunkBytes:        envInt("LOCALAGENT_CHUNK_BYTES", 4096),
		ChunkOverlap:      envInt("LOCALAGENT_CHUNK_OVERLAP", 256),
		SearchHitLimit:    envInt("LOCALAGENT_SEARCH_HIT_LIMIT", 8),
		ContextChunkLimit: envInt("LOCALAGENT_CONTEXT_CHUNK_LIMIT", 4),
		AutoIndexOnStart:  envBool("LOCALAGENT_AUTO_INDEX_ON_START", true),
		RequestTimeout:    envDuration("LOCALAGENT_REQUEST_TIMEOUT", 90*time.Second),
	}
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}
