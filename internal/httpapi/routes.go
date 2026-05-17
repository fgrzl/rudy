package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/fgrzl/localagent/internal/application"
	appcommands "github.com/fgrzl/localagent/internal/application/commands"
	appqueries "github.com/fgrzl/localagent/internal/application/queries"
	"github.com/fgrzl/localagent/internal/config"
	"github.com/fgrzl/mux"
)

type API struct {
	svc    application.Executor
	cfg    config.Config
	log    *slog.Logger
	client *http.Client
}

func New(svc application.Executor, cfg config.Config, log *slog.Logger) *API {
	if log == nil {
		log = slog.Default()
	}
	timeout := cfg.RequestTimeout
	if timeout < 0 {
		timeout = 0
	}
	return &API{
		svc:    svc,
		cfg:    cfg,
		log:    log,
		client: &http.Client{Timeout: timeout},
	}
}

func (api *API) Register(router *mux.Router) {
	if router == nil || api == nil || api.svc == nil {
		return
	}

	router.GET("/healthz", func(c mux.RouteContext) {
		c.OK(map[string]any{"ok": true})
	})

	apiGroup := router.Group("/api")
	apiGroup.GET("/search", api.searchHandler())
	apiGroup.POST("/search", api.searchHandler())
	apiGroup.POST("/index/rebuild", func(c mux.RouteContext) {
		cmd := appcommands.Rebuild{}
		summary, err := cmd.Execute(routeContext(c), api.svc)
		if err != nil {
			c.ServerError("Reindex failed", err.Error())
			return
		}
		c.OK(summary)
	})
	apiGroup.POST("/index/file", func(c mux.RouteContext) {
		var cmd appcommands.IndexFile
		if err := c.Bind(&cmd); err != nil {
			c.BadRequest("Invalid request", err.Error())
			return
		}
		summary, err := cmd.Execute(routeContext(c), api.svc)
		if err != nil {
			c.BadRequest("Invalid request", err.Error())
			return
		}
		c.OK(summary)
	})

	v1 := router.Group("/v1")
	v1.GET("/models", api.proxyHandler("/v1/models"))
	v1.POST("/embeddings", api.proxyHandler("/v1/embeddings"))
	v1.POST("/chat/completions", api.chatCompletionHandler())
}

func (api *API) searchHandler() mux.HandlerFunc {
	return func(c mux.RouteContext) {
		cmd := appqueries.Search{}
		if c.Request() != nil && c.Request().Method == http.MethodPost {
			if err := c.Bind(&cmd); err != nil {
				c.BadRequest("Invalid request", err.Error())
				return
			}
		} else {
			cmd.Query, _ = c.Query().String("q")
			if qLimit, ok := c.Query().Int("limit"); ok {
				cmd.Limit = qLimit
			}
		}

		response, err := cmd.Execute(routeContext(c), api.svc)
		if err != nil {
			c.BadRequest("Invalid request", err.Error())
			return
		}

		c.OK(response)
	}
}

func (api *API) chatCompletionHandler() mux.HandlerFunc {
	return func(c mux.RouteContext) {
		req := c.Request()
		if req == nil {
			c.ServerError("Chat completion failed", "missing request")
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			c.BadRequest("Invalid request", err.Error())
			return
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			c.BadRequest("Invalid request", err.Error())
			return
		}

		if model := strings.TrimSpace(asString(payload["model"])); model == "" {
			payload["model"] = api.cfg.ChatModel
		}

		if rawMessages, ok := payload["messages"]; ok {
			messages, err := decodeChatMessages(rawMessages)
			if err == nil && len(messages) > 0 {
				cmd := appcommands.ChatCompletion{Messages: messages}
				if augmented, err := cmd.Execute(req.Context(), api.svc); err == nil && len(augmented) > 0 {
					payload["messages"] = augmented
				}
			}
		}

		augmented, err := json.Marshal(payload)
		if err != nil {
			c.ServerError("Chat completion failed", err.Error())
			return
		}

		if err := forwardRequest(c.Response(), req, api.client, api.cfg.OllamaBaseURL, "/v1/chat/completions", augmented); err != nil {
			api.log.Error("ollama chat proxy failed", "err", err)
			http.Error(c.Response(), err.Error(), http.StatusBadGateway)
		}
	}
}

func (api *API) proxyHandler(path string) mux.HandlerFunc {
	return func(c mux.RouteContext) {
		req := c.Request()
		if req == nil {
			c.ServerError("Ollama proxy failed", "missing request")
			return
		}
		if err := forwardRequest(c.Response(), req, api.client, api.cfg.OllamaBaseURL, path, nil); err != nil {
			api.log.Error("ollama proxy failed", "path", path, "err", err)
			http.Error(c.Response(), err.Error(), http.StatusBadGateway)
		}
	}
}

func forwardRequest(w http.ResponseWriter, r *http.Request, client *http.Client, baseURL, path string, overrideBody []byte) error {
	var body io.Reader
	if overrideBody != nil {
		body = bytes.NewReader(overrideBody)
	} else if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	target, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), body)
	if err != nil {
		return err
	}
	copyHeaders(req.Header, r.Header)
	req.Header.Del("Host")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	return err
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func decodeChatMessages(raw any) ([]application.ChatMessage, error) {
	rawMessages, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var messages []application.ChatMessage
	if err := json.Unmarshal(rawMessages, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func routeContext(c mux.RouteContext) context.Context {
	if req := c.Request(); req != nil {
		return req.Context()
	}
	return context.Background()
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}
