package commands

import (
	"context"
	"reflect"
	"testing"

	agentpkg "github.com/fgrzl/localagent/internal/agent"
	"github.com/fgrzl/localagent/internal/application"
)

type chatCompletionFakeExecutor struct {
	messages []application.ChatMessage
}

func (f *chatCompletionFakeExecutor) AugmentChatMessages(ctx context.Context, messages []application.ChatMessage) ([]application.ChatMessage, error) {
	f.messages = append([]application.ChatMessage(nil), messages...)
	return append([]application.ChatMessage{{Role: "system", Content: "context"}}, messages...), nil
}

func (f *chatCompletionFakeExecutor) Rebuild(ctx context.Context) (agentpkg.Summary, error) {
	return agentpkg.Summary{}, nil
}

func (f *chatCompletionFakeExecutor) IndexFile(ctx context.Context, absPath string) (agentpkg.Summary, error) {
	return agentpkg.Summary{}, nil
}

func TestChatCompletionExecute(t *testing.T) {
	input := []application.ChatMessage{{Role: "user", Content: "hello"}}
	executor := &chatCompletionFakeExecutor{}

	output, err := ChatCompletion{Messages: input}.Execute(context.Background(), executor)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !reflect.DeepEqual(executor.messages, input) {
		t.Fatalf("messages = %#v, want %#v", executor.messages, input)
	}
	if len(output) != 2 || output[0].Role != "system" {
		t.Fatalf("output = %#v, want augmented messages", output)
	}
}
