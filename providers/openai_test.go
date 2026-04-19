package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-skill-eval-go/tool"
)

func TestOpenAIClientBuildsRequestAndParsesTextResponse(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotRequest openAIChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	client := OpenAIClient{Config: Config{Model: "gpt-test", BaseURL: server.URL + "/v1/", APIKey: "secret-key"}}
	got, err := client.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
		Tools:    ToolDefinitionsFromTools([]tool.Tool{tool.FinishTool{}}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if gotAuth != "Bearer secret-key" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotRequest.Model != "gpt-test" {
		t.Fatalf("unexpected model: %q", gotRequest.Model)
	}
	if len(gotRequest.Messages) != 1 || gotRequest.Messages[0].Content != "hi" {
		t.Fatalf("unexpected messages: %+v", gotRequest.Messages)
	}
	if len(gotRequest.Tools) != 1 || gotRequest.Tools[0].Function.Name != "finish" {
		t.Fatalf("unexpected tools: %+v", gotRequest.Tools)
	}
	parameters := gotRequest.Tools[0].Function.Parameters
	if parameters["type"] != "object" {
		t.Fatalf("unexpected parameters root: %+v", parameters)
	}
	properties := parameters["properties"].(map[string]any)
	required := parameters["required"].([]any)
	if _, ok := properties["final_answer"]; !ok {
		t.Fatalf("expected final_answer property, got %+v", properties)
	}
	if len(required) != 1 || required[0] != "final_answer" {
		t.Fatalf("unexpected required fields: %+v", required)
	}
	if got.Finish == nil || got.Finish.FinalAnswer != "hello" {
		t.Fatalf("unexpected finish response: %+v", got)
	}
}

func TestOpenAIClientParsesToolCallResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call-1","type":"function","function":{"name":"filesystem.write_file","arguments":"{\"path\":\"out.txt\",\"content\":\"ok\"}"}}]},"finish_reason":"tool_calls"}]}`))
	}))
	defer server.Close()

	client := OpenAIClient{Config: Config{Model: "gpt-test", BaseURL: server.URL, APIKey: "secret"}}
	got, err := client.ChatCompletion(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.ToolCalls) != 1 {
		t.Fatalf("unexpected tool calls: %+v", got.ToolCalls)
	}
	call := got.ToolCalls[0]
	if call.ToolName != "filesystem" || call.Operation != "write_file" {
		t.Fatalf("unexpected mapped call: %+v", call)
	}
	if call.Input["path"] != "out.txt" || call.Input["content"] != "ok" {
		t.Fatalf("unexpected call input: %+v", call.Input)
	}
}

func TestOpenAIClientParsesErrorResponseWithoutLeakingKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad auth","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	client := OpenAIClient{Config: Config{Model: "gpt-test", BaseURL: server.URL, APIKey: "super-secret-key"}}
	_, err := client.ChatCompletion(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatalf("expected provider error")
	}
	if !strings.Contains(err.Error(), "bad auth") {
		t.Fatalf("expected provider message, got %v", err)
	}
	if strings.Contains(err.Error(), "super-secret-key") {
		t.Fatalf("error leaked api key: %v", err)
	}
}

func TestOpenAIClientHTTPErrorPath(t *testing.T) {
	client := OpenAIClient{Config: Config{Model: "gpt-test", BaseURL: "http://127.0.0.1:1", APIKey: "secret", Timeout: 10 * time.Millisecond}}
	_, err := client.ChatCompletion(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatalf("expected request error")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaked api key: %v", err)
	}
}

func TestOpenAIClientInvalidConfig(t *testing.T) {
	client := OpenAIClient{Config: Config{}}
	_, err := client.ChatCompletion(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatalf("expected config error")
	}
	if ErrorClassOf(err) != string(ErrorClassConfig) {
		t.Fatalf("expected config error class, got %q", ErrorClassOf(err))
	}
}

func TestOpenAIClientClassifiesAuthRateLimitAndServerErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		wantClass  ErrorClass
		retryable  bool
	}{
		{name: "auth", statusCode: http.StatusUnauthorized, wantClass: ErrorClassAuth, retryable: false},
		{name: "forbidden", statusCode: http.StatusForbidden, wantClass: ErrorClassAuth, retryable: false},
		{name: "rate-limit", statusCode: http.StatusTooManyRequests, wantClass: ErrorClassRateLimit, retryable: true},
		{name: "server", statusCode: http.StatusBadGateway, wantClass: ErrorClassServer, retryable: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(`{"error":{"message":"classified error"}}`))
			}))
			defer server.Close()

			client := OpenAIClient{Config: Config{Model: "gpt-test", BaseURL: server.URL, APIKey: "secret"}}
			_, err := client.ChatCompletion(context.Background(), ChatRequest{})
			if err == nil {
				t.Fatalf("expected provider error")
			}
			if ErrorClassOf(err) != string(tc.wantClass) {
				t.Fatalf("expected error class %q, got %q", tc.wantClass, ErrorClassOf(err))
			}
			if IsRetryable(err) != tc.retryable {
				t.Fatalf("expected retryable=%v, got %v", tc.retryable, IsRetryable(err))
			}
			if StatusCodeOf(err) != tc.statusCode {
				t.Fatalf("expected status code %d, got %d", tc.statusCode, StatusCodeOf(err))
			}
		})
	}
}

func TestOpenAIClientClassifiesTimeoutError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"slow"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	client := OpenAIClient{Config: Config{Model: "gpt-test", BaseURL: server.URL, APIKey: "secret", Timeout: 10 * time.Millisecond}}
	_, err := client.ChatCompletion(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if ErrorClassOf(err) != string(ErrorClassTimeout) {
		t.Fatalf("expected timeout class, got %q", ErrorClassOf(err))
	}
	if !IsRetryable(err) {
		t.Fatalf("expected timeout to be retryable")
	}
}

func TestOpenAIClientUsesEnvAPIKeyFallback(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer env-secret" {
			t.Fatalf("unexpected auth header %q", got)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	client := OpenAIClient{Config: Config{Model: "gpt-test", BaseURL: server.URL}}
	_, err := client.ChatCompletion(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProviderErrorHelpers(t *testing.T) {
	err := &ProviderError{Class: ErrorClassRateLimit, StatusCode: 429, Message: "rate limited", Retryable: true}
	if ErrorClassOf(err) != string(ErrorClassRateLimit) {
		t.Fatalf("unexpected class: %q", ErrorClassOf(err))
	}
	if !IsRetryable(err) {
		t.Fatalf("expected retryable")
	}
	if StatusCodeOf(err) != 429 {
		t.Fatalf("unexpected status: %d", StatusCodeOf(err))
	}
	if IsRetryable(errors.New("plain")) {
		t.Fatalf("plain error should not be retryable")
	}
}

func TestStubClientReturnsConfiguredResponse(t *testing.T) {
	client := StubClient{Response: ChatResponse{Message: Message{Role: "assistant", Content: "hello"}}}
	got, err := client.ChatCompletion(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Message.Content != "hello" {
		t.Fatalf("unexpected response content: %q", got.Message.Content)
	}
}

func TestToolDefinitionsFromTools(t *testing.T) {
	definitions := ToolDefinitionsFromTools([]tool.Tool{tool.FilesystemTool{}, tool.FinishTool{}})
	if len(definitions) != 2 {
		t.Fatalf("unexpected definitions length: %d", len(definitions))
	}
	if definitions[0].Name != "filesystem" {
		t.Fatalf("unexpected first definition: %+v", definitions[0])
	}
	if len(definitions[0].Required) == 0 {
		t.Fatalf("expected required fields in definition")
	}
}
