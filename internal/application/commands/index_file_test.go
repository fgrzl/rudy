package commands

import (
	"context"
	"testing"

	agentpkg "github.com/fgrzl/localagent/internal/agent"
	"github.com/fgrzl/localagent/internal/application"
)

type indexFileFakeExecutor struct {
	path    string
	summary agentpkg.Summary
}

func (f *indexFileFakeExecutor) IndexFile(ctx context.Context, absPath string) (agentpkg.Summary, error) {
	f.path = absPath
	return f.summary, nil
}

func (f *indexFileFakeExecutor) Rebuild(ctx context.Context) (agentpkg.Summary, error) {
	return agentpkg.Summary{}, nil
}

func (f *indexFileFakeExecutor) AugmentChatMessages(ctx context.Context, messages []application.ChatMessage) ([]application.ChatMessage, error) {
	return messages, nil
}

func TestIndexFileExecute(t *testing.T) {
	executor := &indexFileFakeExecutor{summary: agentpkg.Summary{ChunksIndexed: 2}}

	summary, err := IndexFile{Path: "  /workspace/file.go  "}.Execute(context.Background(), executor)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if executor.path != "/workspace/file.go" {
		t.Fatalf("path = %q, want %q", executor.path, "/workspace/file.go")
	}
	if summary != executor.summary {
		t.Fatalf("summary = %#v, want %#v", summary, executor.summary)
	}
}
