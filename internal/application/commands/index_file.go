package commands

import (
	"context"
	"errors"
	"strings"

	agentpkg "github.com/fgrzl/localagent/internal/agent"
)

type IndexFile struct {
	Path string `json:"path"`
}

func (cmd IndexFile) Execute(ctx context.Context, deps Executor) (agentpkg.Summary, error) {
	path := strings.TrimSpace(cmd.Path)
	if path == "" {
		return agentpkg.Summary{}, errors.New("path is required")
	}
	if deps == nil {
		return agentpkg.Summary{}, errors.New("executor is not configured")
	}
	return deps.IndexFile(ctx, path)
}
