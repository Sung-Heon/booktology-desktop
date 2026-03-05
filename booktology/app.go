package main

import (
	"context"
	"fmt"
	"os/exec"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicopt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openaiopt "github.com/openai/openai-go/option"
)

// AIProvider 인터페이스 - 모든 프로바이더가 구현해야 함
type AIProvider interface {
	Analyze(ctx context.Context, topic string, explanation string) (string, error)
}

// ─── Claude CLI 프로바이더 ───────────────────────────────
type ClaudeCLIProvider struct{}

func (p *ClaudeCLIProvider) Analyze(ctx context.Context, topic string, explanation string) (string, error) {
	prompt := buildPrompt(topic, explanation)
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude CLI 실행 실패: %w", err)
	}
	return string(output), nil
}

// ─── Anthropic API 프로바이더 ────────────────────────────
type AnthropicAPIProvider struct {
	client *anthropic.Client
}

func NewAnthropicAPIProvider(apiKey string) *AnthropicAPIProvider {
	client := anthropic.NewClient(anthropicopt.WithAPIKey(apiKey))
	return &AnthropicAPIProvider{client: &client}
}

func (p *AnthropicAPIProvider) Analyze(ctx context.Context, topic string, explanation string) (string, error) {
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5HaikuLatest,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buildPrompt(topic, explanation))),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Anthropic API 호출 실패: %w", err)
	}
	return msg.Content[0].Text, nil
}

// ─── OpenAI API Key 프로바이더 ──────────────────────────
type OpenAIProvider struct {
	client *openai.Client
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	client := openai.NewClient(openaiopt.WithAPIKey(apiKey))
	return &OpenAIProvider{client: &client}
}

func (p *OpenAIProvider) Analyze(ctx context.Context, topic string, explanation string) (string, error) {
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(buildPrompt(topic, explanation)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI API 호출 실패: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}

// ─── ChatGPT OAuth 프로바이더 ────────────────────────────
type ChatGPTOAuthProvider struct {
	token  *OAuthToken
	client *openai.Client
}

func NewChatGPTOAuthProvider(token *OAuthToken) *ChatGPTOAuthProvider {
	client := openai.NewClient(openaiopt.WithAPIKey(token.AccessToken))
	return &ChatGPTOAuthProvider{token: token, client: &client}
}

func (p *ChatGPTOAuthProvider) Analyze(ctx context.Context, topic string, explanation string) (string, error) {
	// 토큰 만료 시 자동 갱신
	if p.token.IsExpired() {
		newToken, err := refreshOAuthToken(ctx, p.token.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("토큰 갱신 실패: %w", err)
		}
		p.token = newToken
		client := openai.NewClient(openaiopt.WithAPIKey(newToken.AccessToken))
		p.client = &client
	}

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(buildPrompt(topic, explanation)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("ChatGPT 호출 실패: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}

// ─── 공통 프롬프트 ───────────────────────────────────────
func buildPrompt(topic string, explanation string) string {
	return fmt.Sprintf(`학생이 "%s"라는 개념을 아래와 같이 설명했습니다.

---
%s
---

파인만 학습법 튜터로서 다음을 분석해주세요:
1. 잘 이해한 부분
2. 이해 갭 또는 틀린 부분
3. 보완이 필요한 핵심 질문 2-3개

한국어로 친절하게 답변해주세요.`, topic, explanation)
}

// ─── App ─────────────────────────────────────────────────
type App struct {
	ctx      context.Context
	provider AIProvider
	config   *AppConfig
}

func NewApp() *App {
	return &App{provider: &ClaudeCLIProvider{}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 저장된 설정 로드
	cfg, err := loadConfig()
	if err != nil {
		cfg = &AppConfig{ProviderType: "claude-cli"}
	}
	a.config = cfg

	// 저장된 프로바이더로 초기화
	switch cfg.ProviderType {
	case "anthropic":
		if cfg.APIKey != "" {
			a.provider = NewAnthropicAPIProvider(cfg.APIKey)
		}
	case "openai":
		if cfg.APIKey != "" {
			a.provider = NewOpenAIProvider(cfg.APIKey)
		}
	case "chatgpt-oauth":
		if cfg.OAuthToken != nil {
			token := &OAuthToken{
				AccessToken:  cfg.OAuthToken.AccessToken,
				RefreshToken: cfg.OAuthToken.RefreshToken,
				ExpiresAt:    cfg.OAuthToken.ExpiresAt,
			}
			a.provider = NewChatGPTOAuthProvider(token)
		}
	}
}

// GetProviderType - 현재 활성 프로바이더 반환 (React UI용)
func (a *App) GetProviderType() string {
	if a.config == nil {
		return "claude-cli"
	}
	return a.config.ProviderType
}

// SetProvider - React에서 프로바이더 설정 시 호출
func (a *App) SetProvider(providerType string, apiKey string) error {
	switch providerType {
	case "claude-cli":
		a.provider = &ClaudeCLIProvider{}
	case "anthropic":
		if apiKey == "" {
			return fmt.Errorf("Anthropic API 키가 필요합니다")
		}
		a.provider = NewAnthropicAPIProvider(apiKey)
	case "openai":
		if apiKey == "" {
			return fmt.Errorf("OpenAI API 키가 필요합니다")
		}
		a.provider = NewOpenAIProvider(apiKey)
	default:
		return fmt.Errorf("알 수 없는 프로바이더: %s", providerType)
	}

	a.config = &AppConfig{ProviderType: providerType, APIKey: apiKey}
	return saveConfig(a.config)
}

// ConnectChatGPTOAuth - 브라우저 OAuth 시작 (React에서 호출)
func (a *App) ConnectChatGPTOAuth() error {
	token, err := StartOpenAIOAuth(a.ctx)
	if err != nil {
		return err
	}
	a.provider = NewChatGPTOAuthProvider(token)
	a.config = &AppConfig{
		ProviderType: "chatgpt-oauth",
		OAuthToken: &SavedToken{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresAt:    token.ExpiresAt,
		},
	}
	return saveConfig(a.config)
}

// AnalyzeExplanation - React Step3에서 호출
func (a *App) AnalyzeExplanation(topic string, explanation string) (string, error) {
	return a.provider.Analyze(a.ctx, topic, explanation)
}
