package commands

import (
	"context"
	"testing"

	agentpkg "github.com/fgrzl/localagent/internal/agent"
	"github.com/fgrzl/localagent/internal/application"
)

type rebuildFakeExecutor struct {
	summary agentpkg.Summary
}

func (f *rebuildFakeExecutor) Rebuild(ctx context.Context) (agentpkg.Summary, error) {
	return f.summary, nil
}

func (f *rebuildFakeExecutor) IndexFile(ctx context.Context, absPath string) (agentpkg.Summary, error) {
	return agentpkg.Summary{}, nil
}

func (f *rebuildFakeExecutor) AugmentChatMessages(ctx context.Context, messages []application.ChatMessage) ([]application.ChatMessage, error) {
	return messages, nil
}

func TestRebuildExecute(t *testing.T) {
	executor := &rebuildFakeExecutor{summary: agentpkg.Summary{FilesIndexed: 3}}

	summary, err := Rebuild{}.Execute(context.Background(), executor)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if summary != executor.summary {
		t.Fatalf("summary = %#v, want %#v", summary, executor.summary)
	}
}
