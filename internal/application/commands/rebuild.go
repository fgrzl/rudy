package commands

import (
	"context"
	"errors"

	agentpkg "github.com/fgrzl/localagent/internal/agent"
	"github.com/fgrzl/localagent/internal/application"
)

type Executor interface {
	Rebuild(ctx context.Context) (agentpkg.Summary, error)
	IndexFile(ctx context.Context, absPath string) (agentpkg.Summary, error)
	AugmentChatMessages(ctx context.Context, messages []application.ChatMessage) ([]application.ChatMessage, error)
}

type Rebuild struct{}

func (Rebuild) Execute(ctx context.Context, deps Executor) (agentpkg.Summary, error) {
	if deps == nil {
		return agentpkg.Summary{}, errors.New("executor is not configured")
	}
	return deps.Rebuild(ctx)
}
