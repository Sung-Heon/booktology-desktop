package main

import (
	"context"
	"fmt"
	"net/http"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicopt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openaiopt "github.com/openai/openai-go/option"
)

// ─── OAuth 토큰 관리 (공통) ──────────────────────────────
type tokenStore struct {
	token     *OAuthToken
	refreshFn func(context.Context, string) (*OAuthToken, error)
}

func (s *tokenStore) ensureFresh(ctx context.Context) error {
	if !s.token.IsExpired() {
		return nil
	}
	t, err := s.refreshFn(ctx, s.token.RefreshToken)
	if err != nil {
		return err
	}
	s.token = t
	return nil
}

// ─── Claude OAuth 프로바이더 ─────────────────────────────
type claudeOAuthTransport struct {
	token string
}

func (t *claudeOAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Del("x-api-key")
	r.Header.Set("Authorization", "Bearer "+t.token)
	r.Header.Set("anthropic-beta", claudeOAuthBeta)
	return http.DefaultTransport.RoundTrip(r)
}

type ClaudeOAuthProvider struct {
	tokenStore
	client *anthropic.Client
	model  string
}

func NewClaudeOAuthProvider(token *OAuthToken, model string) *ClaudeOAuthProvider {
	if model == "" {
		model = string(anthropic.ModelClaude3_5HaikuLatest)
	}
	p := &ClaudeOAuthProvider{
		tokenStore: tokenStore{token: token, refreshFn: refreshClaudeOAuthToken},
		model:      model,
	}
	p.rebuildClient()
	return p
}

func (p *ClaudeOAuthProvider) rebuildClient() {
	httpClient := &http.Client{Transport: &claudeOAuthTransport{token: p.token.AccessToken}}
	client := anthropic.NewClient(anthropicopt.WithAPIKey(""), anthropicopt.WithHTTPClient(httpClient))
	p.client = &client
}

func (p *ClaudeOAuthProvider) Analyze(ctx context.Context, topic string, explanation string, language string) (string, error) {
	prev := p.token.AccessToken
	if err := p.ensureFresh(ctx); err != nil {
		return "", err
	}
	if p.token.AccessToken != prev {
		p.rebuildClient()
	}
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buildPrompt(topic, explanation, language))),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Claude OAuth API 호출 실패: %w", err)
	}
	return msg.Content[0].Text, nil
}

func (p *ClaudeOAuthProvider) Chat(ctx context.Context, history []ChatMessage, message string) (string, error) {
	prev := p.token.AccessToken
	if err := p.ensureFresh(ctx); err != nil {
		return "", err
	}
	if p.token.AccessToken != prev {
		p.rebuildClient()
	}
	params := make([]anthropic.MessageParam, 0, len(history))
	for _, h := range history {
		if h.Role == "user" {
			params = append(params, anthropic.NewUserMessage(anthropic.NewTextBlock(h.Content)))
		} else {
			params = append(params, anthropic.NewAssistantMessage(anthropic.NewTextBlock(h.Content)))
		}
	}
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 1024,
		Messages:  params,
	})
	if err != nil {
		return "", fmt.Errorf("Claude OAuth 채팅 실패: %w", err)
	}
	return msg.Content[0].Text, nil
}

// ─── Anthropic API 프로바이더 ────────────────────────────
type AnthropicAPIProvider struct {
	client *anthropic.Client
	model  string
}

func NewAnthropicAPIProvider(apiKey string, model string) *AnthropicAPIProvider {
	client := anthropic.NewClient(anthropicopt.WithAPIKey(apiKey))
	if model == "" {
		model = string(anthropic.ModelClaude3_5HaikuLatest)
	}
	return &AnthropicAPIProvider{client: &client, model: model}
}

func (p *AnthropicAPIProvider) Analyze(ctx context.Context, topic string, explanation string, language string) (string, error) {
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buildPrompt(topic, explanation, language))),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Anthropic API 호출 실패: %w", err)
	}
	return msg.Content[0].Text, nil
}

func (p *AnthropicAPIProvider) Chat(ctx context.Context, history []ChatMessage, message string) (string, error) {
	params := make([]anthropic.MessageParam, 0, len(history)+1)
	for _, h := range history {
		if h.Role == "user" {
			params = append(params, anthropic.NewUserMessage(anthropic.NewTextBlock(h.Content)))
		} else {
			params = append(params, anthropic.NewAssistantMessage(anthropic.NewTextBlock(h.Content)))
		}
	}
	params = append(params, anthropic.NewUserMessage(anthropic.NewTextBlock(message)))
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 1024,
		Messages:  params,
	})
	if err != nil {
		return "", fmt.Errorf("Anthropic API 호출 실패: %w", err)
	}
	return msg.Content[0].Text, nil
}

// ─── OpenAI API 프로바이더 ───────────────────────────────
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

func NewOpenAIProvider(apiKey string, model string) *OpenAIProvider {
	client := openai.NewClient(openaiopt.WithAPIKey(apiKey))
	if model == "" {
		model = string(openai.ChatModelGPT4oMini)
	}
	return &OpenAIProvider{client: &client, model: model}
}

func (p *OpenAIProvider) Analyze(ctx context.Context, topic string, explanation string, language string) (string, error) {
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(p.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(buildPrompt(topic, explanation, language)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI API 호출 실패: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, history []ChatMessage, message string) (string, error) {
	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(history)+1)
	for _, h := range history {
		if h.Role == "user" {
			msgs = append(msgs, openai.UserMessage(h.Content))
		} else {
			msgs = append(msgs, openai.AssistantMessage(h.Content))
		}
	}
	msgs = append(msgs, openai.UserMessage(message))
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(p.model),
		Messages: msgs,
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI API 호출 실패: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}

// ─── ChatGPT OAuth 프로바이더 ────────────────────────────
type ChatGPTOAuthProvider struct {
	tokenStore
	client *openai.Client
	model  string
}

func NewChatGPTOAuthProvider(token *OAuthToken, model string) *ChatGPTOAuthProvider {
	if model == "" {
		model = "gpt-4o"
	}
	p := &ChatGPTOAuthProvider{
		tokenStore: tokenStore{token: token, refreshFn: refreshOAuthToken},
		model:      model,
	}
	p.rebuildClient()
	return p
}

func (p *ChatGPTOAuthProvider) rebuildClient() {
	client := openai.NewClient(
		openaiopt.WithAPIKey(p.token.AccessToken),
		openaiopt.WithBaseURL("https://api.openai.com/v1/"),
		openaiopt.WithHeader("Authorization", "Bearer "+p.token.AccessToken),
	)
	p.client = &client
}

func (p *ChatGPTOAuthProvider) Analyze(ctx context.Context, topic string, explanation string, language string) (string, error) {
	prev := p.token.AccessToken
	if err := p.ensureFresh(ctx); err != nil {
		return "", err
	}
	if p.token.AccessToken != prev {
		p.rebuildClient()
	}
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(p.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(buildPrompt(topic, explanation, language)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("ChatGPT 호출 실패: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}

func (p *ChatGPTOAuthProvider) Chat(ctx context.Context, history []ChatMessage, message string) (string, error) {
	prev := p.token.AccessToken
	if err := p.ensureFresh(ctx); err != nil {
		return "", err
	}
	if p.token.AccessToken != prev {
		p.rebuildClient()
	}
	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(history)+1)
	for _, h := range history {
		if h.Role == "user" {
			msgs = append(msgs, openai.UserMessage(h.Content))
		} else {
			msgs = append(msgs, openai.AssistantMessage(h.Content))
		}
	}
	msgs = append(msgs, openai.UserMessage(message))
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(p.model),
		Messages: msgs,
	})
	if err != nil {
		return "", fmt.Errorf("ChatGPT 호출 실패: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}
