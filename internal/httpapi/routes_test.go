package httpapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMergeModelCatalog(t *testing.T) {
	body := []byte(`{"object":"list","data":[{"id":"qwen2.5-coder:7b-instruct","object":"model","owned_by":"ollama"}]}`)

	merged, err := mergeModelCatalog(body, []string{
		"qwen2.5-coder:7b-instruct",
		"qwen2.5-coder:14b-instruct",
		"llama3.1:8b-instruct",
		"qwen2.5-coder:14b-instruct",
	})
	if err != nil {
		t.Fatalf("mergeModelCatalog returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatalf("unmarshal merged payload: %v", err)
	}

	if got := payload["object"]; got != "list" {
		t.Fatalf("object = %v, want %q", got, "list")
	}

	rawData, ok := payload["data"]
	if !ok {
		t.Fatalf("merged payload missing data")
	}

	data, ok := rawData.([]any)
	if !ok {
		t.Fatalf("data has type %T, want []any", rawData)
	}

	ids := make([]string, 0, len(data))
	for _, item := range data {
		model, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("model entry has type %T, want map[string]any", item)
		}
		id, ok := model["id"].(string)
		if !ok {
			t.Fatalf("model entry missing string id: %#v", model)
		}
		ids = append(ids, id)
	}

	want := []string{
		"qwen2.5-coder:7b-instruct",
		"qwen2.5-coder:14b-instruct",
		"llama3.1:8b-instruct",
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("ids = %#v, want %#v", ids, want)
	}
}
