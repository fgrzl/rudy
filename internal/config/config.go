package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr            string
	OllamaBaseURL       string
	ChatModel           string
	SupportedChatModels []string
	EmbeddingModel      string
	DataDir             string
	StoreDir            string
	ManifestPath        string
	WorkspaceDir        string
	IndexName           string
	ChunkBytes          int
	ChunkOverlap        int
	SearchHitLimit      int
	ContextChunkLimit   int
	AutoIndexOnStart    bool
	RequestTimeout      time.Duration
}

func Load() Config {
	dataDir := envString("LOCALAGENT_DATA_DIR", "/data")
	storeDir := filepath.Join(dataDir, "pebble")
	chatModel := envString("LOCALAGENT_CHAT_MODEL", "qwen2.5-coder:7b-instruct")
	supportedChatModels := envStrings("LOCALAGENT_SUPPORTED_CHAT_MODELS", []string{
		"qwen2.5-coder:7b-instruct",
		"qwen2.5-coder:14b-instruct",
		"qwen2.5-coder:32b-instruct",
		"qwen3-coder:30b-a3b-instruct",
		"deepseek-coder-v2:16b-lite-instruct",
		"llama3.1:8b-instruct",
	})
	supportedChatModels = uniqueStrings(append([]string{chatModel}, supportedChatModels...))

	return Config{
		HTTPAddr:            envString("LOCALAGENT_HTTP_ADDR", ":8080"),
		OllamaBaseURL:       envString("LOCALAGENT_OLLAMA_URL", "http://ollama:11434"),
		ChatModel:           chatModel,
		SupportedChatModels: supportedChatModels,
		EmbeddingModel:      envString("LOCALAGENT_EMBEDDING_MODEL", "nomic-embed-text"),
		DataDir:             dataDir,
		StoreDir:            storeDir,
		ManifestPath:        filepath.Join(dataDir, "index-manifest.json"),
		WorkspaceDir:        envString("LOCALAGENT_WORKSPACE_DIR", "/workspace"),
		IndexName:           envString("LOCALAGENT_INDEX_NAME", "workspace"),
		ChunkBytes:          envInt("LOCALAGENT_CHUNK_BYTES", 4096),
		ChunkOverlap:        envInt("LOCALAGENT_CHUNK_OVERLAP", 256),
		SearchHitLimit:      envInt("LOCALAGENT_SEARCH_HIT_LIMIT", 8),
		ContextChunkLimit:   envInt("LOCALAGENT_CONTEXT_CHUNK_LIMIT", 4),
		AutoIndexOnStart:    envBool("LOCALAGENT_AUTO_INDEX_ON_START", true),
		RequestTimeout:      envDuration("LOCALAGENT_REQUEST_TIMEOUT", 90*time.Second),
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

func envStrings(key string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		items := strings.Split(value, ",")
		result := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				result = append(result, item)
			}
		}
		if len(result) > 0 {
			return uniqueStrings(result)
		}
	}
	return uniqueStrings(append([]string(nil), fallback...))
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
