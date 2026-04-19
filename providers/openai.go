// Package providers contains Phase 1 migration provider contracts and OpenAI-compatible HTTP support.
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"agent-skill-eval-go/tool"
)

var ErrNotImplemented = errors.New("provider request not implemented")

var ErrInvalidConfig = errors.New("invalid provider config")

// Config contains minimal OpenAI-compatible provider settings.
type Config struct {
	Model        string        `json:"model"`
	BaseURL      string        `json:"base_url"`
	APIKey       string        `json:"-"`
	Timeout      time.Duration `json:"timeout,omitempty"`
	MaxRetries   int           `json:"max_retries,omitempty"`
	RetryBackoff time.Duration `json:"retry_backoff,omitempty"`
}

// Message is a provider-neutral chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ToolCall describes one provider-requested tool invocation.
type ToolCall struct {
	ID        string         `json:"id,omitempty"`
	ToolName  string         `json:"tool_name"`
	Operation string         `json:"operation,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
}

// FinishSignal describes an explicit provider-side finish response.
type FinishSignal struct {
	FinalAnswer string `json:"final_answer"`
	Reason      string `json:"reason,omitempty"`
}

// ChatRequest is the minimal chat/completion style request shape.
type ChatRequest struct {
	Messages []Message        `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
}

// ToolDefinition is a minimal OpenAI-compatible function tool definition.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Required    []string       `json:"required,omitempty"`
}

// ChatResponse is the minimal chat/completion style response shape.
type ChatResponse struct {
	Message   Message       `json:"message"`
	ToolCalls []ToolCall    `json:"tool_calls,omitempty"`
	Finish    *FinishSignal `json:"finish,omitempty"`
}

// ChatClient is the future provider contract used by the new architecture.
type ChatClient interface {
	ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// OpenAIClient is a minimal OpenAI-compatible provider client.
type OpenAIClient struct {
	Config     Config
	HTTPClient *http.Client
}

func (c OpenAIClient) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	config, err := c.normalizedConfig()
	if err != nil {
		return ChatResponse{}, &ProviderError{Class: ErrorClassConfig, Message: err.Error(), Retryable: false, Temporary: false, Cause: err}
	}

	body, err := json.Marshal(openAIChatRequest{
		Model:    config.Model,
		Messages: toOpenAIMessages(req.Messages),
		Tools:    toOpenAITools(req.Tools),
	})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("encode provider request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsURL(config.BaseURL), bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create provider request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: config.Timeout}
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, classifyTransportError(err)
	}
	defer httpResp.Body.Close()

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return ChatResponse{}, &ProviderError{Class: ErrorClassBadResponse, Message: "failed to read provider response", Retryable: false, Temporary: false, Cause: err}
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return ChatResponse{}, parseOpenAIError(httpResp.StatusCode, data)
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return ChatResponse{}, &ProviderError{Class: ErrorClassBadResponse, Message: "failed to parse provider response", Retryable: false, Temporary: false, Cause: err}
	}
	return fromOpenAIResponse(parsed), nil
}

// StubClient is a test-friendly in-memory provider implementation.
type StubClient struct {
	Response ChatResponse
	Err      error
}

func (s StubClient) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if s.Err != nil {
		return ChatResponse{}, s.Err
	}
	return s.Response, nil
}

func (c OpenAIClient) normalizedConfig() (Config, error) {
	config := c.Config
	config.Model = strings.TrimSpace(config.Model)
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	config.APIKey = strings.TrimSpace(config.APIKey)
	if config.APIKey == "" {
		config.APIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = 200 * time.Millisecond
	}
	if config.Model == "" {
		return Config{}, fmt.Errorf("%w: model is required", ErrInvalidConfig)
	}
	if config.BaseURL == "" {
		return Config{}, fmt.Errorf("%w: base_url is required", ErrInvalidConfig)
	}
	if _, err := url.ParseRequestURI(config.BaseURL); err != nil {
		return Config{}, fmt.Errorf("%w: base_url is invalid: %w", ErrInvalidConfig, err)
	}
	if config.APIKey == "" {
		return Config{}, fmt.Errorf("%w: api_key is required", ErrInvalidConfig)
	}
	return config, nil
}

func chatCompletionsURL(baseURL string) string {
	trimmed := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(trimmed, "/chat/completions") {
		return trimmed
	}
	return trimmed + "/chat/completions"
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Tools    []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Message      openAIResponseMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

type openAIResponseMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func toOpenAIMessages(messages []Message) []openAIMessage {
	out := make([]openAIMessage, 0, len(messages))
	for _, message := range messages {
		out = append(out, openAIMessage{Role: message.Role, Content: message.Content})
	}
	return out
}

func toOpenAITools(tools []ToolDefinition) []openAITool {
	out := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters: map[string]any{
					"type":       "object",
					"properties": tool.Parameters,
					"required":   tool.Required,
				},
			},
		})
	}
	return out
}

// ToolDefinitionsFromTools converts internal tool specs into provider tool definitions.
func ToolDefinitionsFromTools(tools []tool.Tool) []ToolDefinition {
	definitions := make([]ToolDefinition, 0, len(tools))
	for _, current := range tools {
		spec := current.Spec()
		properties := make(map[string]any, len(spec.Parameters))
		for name, parameter := range spec.Parameters {
			property := map[string]any{
				"type":        string(parameter.Type),
				"description": parameter.Description,
			}
			if len(parameter.Enum) > 0 {
				property["enum"] = parameter.Enum
			}
			properties[name] = property
		}
		definitions = append(definitions, ToolDefinition{
			Name:        spec.Name,
			Description: spec.Description,
			Parameters:  properties,
			Required:    tool.RequiredFields(spec),
		})
	}
	return definitions
}

func fromOpenAIResponse(response openAIChatResponse) ChatResponse {
	if len(response.Choices) == 0 {
		return ChatResponse{}
	}
	choice := response.Choices[0]
	message := choice.Message
	out := ChatResponse{
		Message: Message{
			Role:    message.Role,
			Content: message.Content,
		},
	}

	for _, call := range message.ToolCalls {
		toolName, operation, input := parseToolFunction(call.Function.Name, call.Function.Arguments)
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        call.ID,
			ToolName:  toolName,
			Operation: operation,
			Input:     input,
		})
	}

	if len(out.ToolCalls) == 0 && strings.TrimSpace(message.Content) != "" && choice.FinishReason == "stop" {
		out.Finish = &FinishSignal{
			FinalAnswer: message.Content,
			Reason:      "stop",
		}
	}
	return out
}

func parseToolFunction(name string, arguments string) (string, string, map[string]any) {
	input := map[string]any{}
	if strings.TrimSpace(arguments) != "" {
		_ = json.Unmarshal([]byte(arguments), &input)
	}

	toolName := name
	operation := ""
	if strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)
		toolName = parts[0]
		operation = parts[1]
	}
	if value, ok := input["tool_name"].(string); ok && value != "" {
		toolName = value
	}
	if value, ok := input["tool"].(string); ok && value != "" {
		toolName = value
	}
	if value, ok := input["operation"].(string); ok && value != "" {
		operation = value
	}
	return toolName, operation, input
}

func parseOpenAIError(status int, data []byte) error {
	var parsed openAIErrorResponse
	if err := json.Unmarshal(data, &parsed); err == nil && parsed.Error.Message != "" {
		switch {
		case status == http.StatusUnauthorized || status == http.StatusForbidden:
			return &ProviderError{Class: ErrorClassAuth, StatusCode: status, Message: parsed.Error.Message, Retryable: false, Temporary: false}
		case status == http.StatusTooManyRequests:
			return &ProviderError{Class: ErrorClassRateLimit, StatusCode: status, Message: parsed.Error.Message, Retryable: true, Temporary: true}
		case status >= 500:
			return &ProviderError{Class: ErrorClassServer, StatusCode: status, Message: parsed.Error.Message, Retryable: true, Temporary: true}
		default:
			return &ProviderError{Class: ErrorClassBadResponse, StatusCode: status, Message: parsed.Error.Message, Retryable: false, Temporary: false}
		}
	}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &ProviderError{Class: ErrorClassAuth, StatusCode: status, Message: "authentication failed", Retryable: false, Temporary: false}
	case status == http.StatusTooManyRequests:
		return &ProviderError{Class: ErrorClassRateLimit, StatusCode: status, Message: "rate limited", Retryable: true, Temporary: true}
	case status >= 500:
		return &ProviderError{Class: ErrorClassServer, StatusCode: status, Message: "provider server error", Retryable: true, Temporary: true}
	default:
		return &ProviderError{Class: ErrorClassBadResponse, StatusCode: status, Message: fmt.Sprintf("provider error status %d", status), Retryable: false, Temporary: false}
	}
}
