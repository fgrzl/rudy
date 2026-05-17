package queries

import (
	"context"
	"errors"
	"strings"

	"github.com/fgrzl/localagent/internal/application"
)

type Executor interface {
	Search(ctx context.Context, query string, limit int) (application.SearchResponse, error)
}

type Search struct {
	Query string `json:"query"`
	Text  string `json:"text"`
	Limit int    `json:"limit"`
}

func (q Search) Execute(ctx context.Context, deps Executor) (application.SearchResponse, error) {
	query := strings.TrimSpace(firstNonEmpty(q.Query, q.Text))
	if query == "" {
		return application.SearchResponse{}, errors.New("query is required")
	}
	if deps == nil {
		return application.SearchResponse{}, errors.New("search executor is not configured")
	}
	return deps.Search(ctx, query, q.Limit)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
