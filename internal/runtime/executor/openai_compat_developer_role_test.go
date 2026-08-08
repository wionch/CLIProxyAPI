package executor

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestNormalizeDeveloperRole(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantRoles []string
	}{
		{name: "no messages", in: `{}`},
		{name: "empty messages", in: `{"messages":[]}`},
		{name: "developer first message", in: `{"messages":[{"role":"developer","content":"sys"},{"role":"user","content":"hi"}]}`, wantRoles: []string{"system", "user"}},
		{name: "developer later message", in: `{"messages":[{"role":"user","content":"hi"},{"role":"developer","content":"sys"}]}`, wantRoles: []string{"user", "system"}},
		{name: "system untouched", in: `{"messages":[{"role":"system","content":"sys"}]}`, wantRoles: []string{"system"}},
		{name: "non-array messages untouched", in: `{"messages":{"role":"developer"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := normalizeDeveloperRole([]byte(tc.in))
			roles := gjson.GetBytes(out, "messages.#.role").Array()
			for _, r := range roles {
				if r.String() == "developer" {
					t.Fatalf("role developer leaked through: %s", string(out))
				}
			}
			if len(tc.wantRoles) > 0 {
				if len(roles) != len(tc.wantRoles) {
					t.Fatalf("got %d roles, want %d (body: %s)", len(roles), len(tc.wantRoles), string(out))
				}
				for i := range tc.wantRoles {
					if roles[i].String() != tc.wantRoles[i] {
						t.Fatalf("messages.%d.role = %q, want %q (body: %s)", i, roles[i].String(), tc.wantRoles[i], string(out))
					}
				}
			}
		})
	}
}

func TestOpenAICompatExecutorRewritesDeveloperRoleUpstream(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`))
	}))
	defer server.Close()

	executor := NewOpenAICompatExecutor("opencode", &config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{
		"base_url": server.URL,
		"api_key":  "test-key",
	}}
	payload := []byte(`{"model":"opencode__minimax-m3","messages":[{"role":"developer","content":"You are a helpful assistant."},{"role":"user","content":"hello"}]}`)
	req := cliproxyexecutor.Request{Model: "opencode__minimax-m3", Payload: payload}
	opts := cliproxyexecutor.Options{Stream: false, SourceFormat: sdktranslator.FormatOpenAI}

	resp, err := executor.Execute(t.Context(), auth, req, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(resp.Payload) == 0 {
		t.Fatal("empty response payload")
	}
	if !strings.Contains(string(gotBody), `"role":"system"`) {
		t.Fatalf("upstream request still contains developer role or missing system role: %s", string(gotBody))
	}
	if strings.Contains(string(gotBody), "developer") {
		t.Fatalf("upstream request leaked developer role: %s", string(gotBody))
	}
}

func TestOpenAICompatExecutorStreamRewritesDeveloperRoleUpstream(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hi\"},\"finish_reason\":null}]}\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()

	executor := NewOpenAICompatExecutor("opencode", &config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{
		"base_url": server.URL,
		"api_key":  "test-key",
	}}
	payload := []byte(`{"model":"opencode__kimi-k3","messages":[{"role":"developer","content":"Be concise."}]}`)
	req := cliproxyexecutor.Request{Model: "opencode__kimi-k3", Payload: payload}
	opts := cliproxyexecutor.Options{Stream: true, SourceFormat: sdktranslator.FormatOpenAI}

	result, err := executor.ExecuteStream(t.Context(), auth, req, opts)
	if err != nil {
		t.Fatalf("ExecuteStream failed: %v", err)
	}
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error: %v", chunk.Err)
		}
	}
	if strings.Contains(string(gotBody), "developer") {
		t.Fatalf("upstream stream request leaked developer role: %s", string(gotBody))
	}
	if !strings.Contains(string(gotBody), `"role":"system"`) {
		t.Fatalf("upstream stream request missing system role: %s", string(gotBody))
	}
}
