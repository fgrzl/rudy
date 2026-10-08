package application

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestChatMessagePreservesOpenAIFields(t *testing.T) {
	for _, input := range []string{
		`{"role":"assistant","content":null,"reasoning_content":"reasoning","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}}]}`,
		`{"role":"tool","content":"file contents","tool_call_id":"call_1","name":"read"}`,
		`{"role":"user","content":[{"type":"text","text":"find the handler"},{"type":"image_url","image_url":{"url":"data:image/png;base64,abc"}}]}`,
	} {
		var msg ChatMessage
		if err := json.Unmarshal([]byte(input), &msg); err != nil {
			t.Fatal(err)
		}
		output, err := json.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		var want, got any
		_ = json.Unmarshal([]byte(input), &want)
		_ = json.Unmarshal(output, &got)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("message fields changed: got %s, want %s", output, input)
		}
	}
}
