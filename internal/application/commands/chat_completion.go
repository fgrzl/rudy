package commands

import (
	"context"
	"errors"

	"github.com/fgrzl/localagent/internal/application"
)

type ChatCompletion struct {
	Messages []application.ChatMessage `json:"messages"`
}

func (cmd ChatCompletion) Execute(ctx context.Context, deps Executor) ([]application.ChatMessage, error) {
	if len(cmd.Messages) == 0 {
		return nil, nil
	}
	if deps == nil {
		return nil, errors.New("executor is not configured")
	}
	return deps.AugmentChatMessages(ctx, cmd.Messages)
}
