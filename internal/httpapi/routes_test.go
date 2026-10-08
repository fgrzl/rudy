package httpapi

import (
	"bufio"
	"encoding/json"
	"github.com/fgrzl/localagent/internal/application"
	"github.com/fgrzl/localagent/internal/config"
	"github.com/fgrzl/mux"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
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

func TestForwardRequestStreamsBeforeUpstreamFinishes(t *testing.T) {
	first := make(chan struct{})
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: first\n\n"))
		w.(http.Flusher).Flush()
		close(first)
		<-release
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = forwardRequest(w, r, upstream.Client(), upstream.URL, "/", nil)
	}))
	defer downstream.Close()
	defer close(release)
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(downstream.URL)
	if err != nil {
		t.Fatalf("stream did not arrive while upstream was open: %v", err)
	}
	defer resp.Body.Close()
	<-first
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil || line != "data: first\n" {
		t.Fatalf("first event = %q, err = %v", line, err)
	}
}

func TestChatRoutePreservesToolHistory(t *testing.T) {
	input := `{"messages":[{"role":"assistant","content":null,"reasoning_content":"inspect file","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"file contents"},{"role":"user","content":[{"type":"text","text":"explain"}]}],"tools":[{"type":"function","function":{"name":"read","parameters":{"type":"object"}}}],"stream":false}`
	var captured map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("upstream path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer upstream.Close()
	cfg := config.Config{OllamaBaseURL: upstream.URL, ChatModel: "local", RequestTimeout: time.Second}
	router := mux.NewRouter()
	if err := router.Configure(New(application.New(nil, cfg), cfg, nil).Register); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(input))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var want map[string]any
	_ = json.Unmarshal([]byte(input), &want)
	want["model"] = "local"
	if !reflect.DeepEqual(captured, want) {
		t.Fatalf("upstream request changed: got %#v, want %#v", captured, want)
	}
}
