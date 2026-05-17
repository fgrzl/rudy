package queries

import (
	"context"
	"reflect"
	"testing"

	"github.com/fgrzl/localagent/internal/application"
)

type searchFakeExecutor struct {
	query  string
	limit  int
	result application.SearchResponse
	called bool
}

func (f *searchFakeExecutor) Search(ctx context.Context, query string, limit int) (application.SearchResponse, error) {
	f.called = true
	f.query = query
	f.limit = limit
	return f.result, nil
}

func TestSearchExecute(t *testing.T) {
	executor := &searchFakeExecutor{result: application.SearchResponse{Query: "fallback", Count: 1}}

	response, err := Search{Text: "  fallback  ", Limit: 7}.Execute(context.Background(), executor)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !executor.called {
		t.Fatal("Search was not called")
	}
	if executor.query != "fallback" {
		t.Fatalf("query = %q, want %q", executor.query, "fallback")
	}
	if executor.limit != 7 {
		t.Fatalf("limit = %d, want %d", executor.limit, 7)
	}
	if !reflect.DeepEqual(response, executor.result) {
		t.Fatalf("response = %#v, want %#v", response, executor.result)
	}
}
