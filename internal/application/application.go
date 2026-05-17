package application

import (
	"context"
	"strings"

	searchoverlay "github.com/fgrzl/kv/pkg/search"
	agentpkg "github.com/fgrzl/localagent/internal/agent"
	"github.com/fgrzl/localagent/internal/config"
	"github.com/fgrzl/localagent/internal/indexer"
)

type Agent interface {
	Search(ctx context.Context, query string, limit int) ([]searchoverlay.SearchHit, error)
	SearchContext(ctx context.Context, query string, limit int) (string, error)
	Rebuild(ctx context.Context) (agentpkg.Summary, error)
	IndexFile(ctx context.Context, absPath string) (agentpkg.Summary, error)
}

type Executor interface {
	Search(ctx context.Context, query string, limit int) (SearchResponse, error)
	Rebuild(ctx context.Context) (agentpkg.Summary, error)
	IndexFile(ctx context.Context, absPath string) (agentpkg.Summary, error)
	AugmentChatMessages(ctx context.Context, messages []ChatMessage) ([]ChatMessage, error)
}

type Application struct {
	agent Agent
	cfg   config.Config
}

func New(agent Agent, cfg config.Config) *Application {
	return &Application{agent: agent, cfg: cfg}
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type SearchResult struct {
	ID            string   `json:"id"`
	Score         float64  `json:"score"`
	MatchedFields []string `json:"matched_fields"`
	Path          string   `json:"path"`
	ChunkIndex    int      `json:"chunk_index"`
	ChunkCount    int      `json:"chunk_count"`
	Content       string   `json:"content"`
}

type SearchResponse struct {
	Query   string         `json:"query"`
	Count   int            `json:"count"`
	Results []SearchResult `json:"results"`
}

func (a *Application) Search(ctx context.Context, query string, limit int) (SearchResponse, error) {
	if limit <= 0 {
		limit = a.cfg.SearchHitLimit
	}
	if a.agent == nil {
		return SearchResponse{}, nil
	}

	hits, err := a.agent.Search(ctx, query, limit)
	if err != nil {
		return SearchResponse{}, err
	}

	results := make([]SearchResult, 0, len(hits))
	for _, hit := range hits {
		result, ok := searchResultFromHit(hit)
		if !ok {
			continue
		}
		results = append(results, result)
	}

	return SearchResponse{
		Query:   query,
		Count:   len(results),
		Results: results,
	}, nil
}

func (a *Application) Rebuild(ctx context.Context) (agentpkg.Summary, error) {
	if a.agent == nil {
		return agentpkg.Summary{}, nil
	}
	return a.agent.Rebuild(ctx)
}

func (a *Application) IndexFile(ctx context.Context, absPath string) (agentpkg.Summary, error) {
	if a.agent == nil {
		return agentpkg.Summary{}, nil
	}
	return a.agent.IndexFile(ctx, absPath)
}

func (a *Application) SearchContext(ctx context.Context, query string) (string, error) {
	if a.agent == nil {
		return "", nil
	}

	limit := a.cfg.ContextChunkLimit
	if limit <= 0 {
		limit = 4
	}

	hits, err := a.agent.Search(ctx, query, limit)
	if err != nil {
		return "", err
	}
	return indexer.BuildChatContext(hits, limit, 12000), nil
}

func (a *Application) AugmentChatMessages(ctx context.Context, messages []ChatMessage) ([]ChatMessage, error) {
	query := lastUserMessage(messages)
	if strings.TrimSpace(query) == "" {
		return messages, nil
	}

	contextText, err := a.SearchContext(ctx, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(contextText) == "" {
		return messages, nil
	}

	augmented := make([]ChatMessage, 0, len(messages)+1)
	augmented = append(augmented, ChatMessage{
		Role:    "system",
		Content: "Use the indexed workspace context below when it is relevant. If it is not relevant, answer normally.\n\n" + contextText,
	})
	augmented = append(augmented, messages...)
	return augmented, nil
}

func searchResultFromHit(hit searchoverlay.SearchHit) (SearchResult, bool) {
	payload, err := indexer.DecodeChunkPayload(hit.Payload)
	if err != nil {
		return SearchResult{}, false
	}
	return SearchResult{
		ID:            hit.ID,
		Score:         hit.Score,
		MatchedFields: hit.MatchedFields,
		Path:          payload.RelativePath,
		ChunkIndex:    payload.ChunkIndex,
		ChunkCount:    payload.ChunkCount,
		Content:       preview(payload.Content, 1200),
	}, true
}

func lastUserMessage(messages []ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if strings.EqualFold(msg.Role, "user") && strings.TrimSpace(msg.Content) != "" {
			return msg.Content
		}
	}
	return ""
}

func preview(content string, limit int) string {
	content = strings.TrimSpace(content)
	if limit <= 0 || len(content) <= limit {
		return content
	}
	return content[:limit] + "..."
}
