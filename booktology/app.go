package main

import (
	"context"
	"fmt"
	"os"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

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

	cfg, err := loadConfig()
	if err != nil || cfg == nil {
		cfg = &AppConfig{ProviderType: "claude-cli", Language: "auto"}
	}
	if cfg.Language == "" {
		cfg.Language = "auto"
	}
	a.config = cfg

	switch cfg.ProviderType {
	case "claude-cli":
		a.provider = &ClaudeCLIProvider{model: cfg.Model}
	case "claude-persistent":
		p := NewClaudePersistentProvider(cfg.Model)
		p.env = removeEnv(os.Environ(), "CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT")
		a.provider = p
	case "anthropic":
		if cfg.APIKey != "" {
			a.provider = NewAnthropicAPIProvider(cfg.APIKey, cfg.Model)
		}
	case "openai":
		if cfg.APIKey != "" {
			a.provider = NewOpenAIProvider(cfg.APIKey, cfg.Model)
		}
	case "chatgpt-oauth":
		if cfg.OAuthToken != nil {
			a.provider = NewChatGPTOAuthProvider(&OAuthToken{
				AccessToken:  cfg.OAuthToken.AccessToken,
				RefreshToken: cfg.OAuthToken.RefreshToken,
				ExpiresAt:    cfg.OAuthToken.ExpiresAt,
			}, cfg.Model)
		}
	case "claude-oauth":
		if cfg.ClaudeOAuthToken != nil {
			a.provider = NewClaudeOAuthProvider(&OAuthToken{
				AccessToken:  cfg.ClaudeOAuthToken.AccessToken,
				RefreshToken: cfg.ClaudeOAuthToken.RefreshToken,
				ExpiresAt:    cfg.ClaudeOAuthToken.ExpiresAt,
			}, cfg.Model)
		}
	}
}

// GetConfig - 현재 설정 반환
func (a *App) GetConfig() AppConfig {
	if a.config == nil {
		return AppConfig{ProviderType: "claude-cli", Language: "auto"}
	}
	return AppConfig{
		ProviderType: a.config.ProviderType,
		Model:        a.config.Model,
		Language:     a.config.Language,
	}
}

// SetProvider - 프로바이더 + API 키 설정
func (a *App) SetProvider(providerType string, apiKey string) error {
	model := ""
	if a.config != nil {
		model = a.config.Model
	}
	switch providerType {
	case "claude-cli":
		a.provider = &ClaudeCLIProvider{model: model}
	case "claude-persistent":
		p := NewClaudePersistentProvider(model)
		p.env = removeEnv(os.Environ(), "CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT")
		a.provider = p
	case "codex-cli":
		a.provider = &CodexCLIProvider{model: model}
	case "anthropic":
		if apiKey == "" {
			return fmt.Errorf("Anthropic API 키가 필요합니다")
		}
		a.provider = NewAnthropicAPIProvider(apiKey, model)
	case "openai":
		if apiKey == "" {
			return fmt.Errorf("OpenAI API 키가 필요합니다")
		}
		a.provider = NewOpenAIProvider(apiKey, model)
	case "chatgpt-oauth":
		if a.config != nil && a.config.OAuthToken != nil {
			a.provider = NewChatGPTOAuthProvider(&OAuthToken{
				AccessToken:  a.config.OAuthToken.AccessToken,
				RefreshToken: a.config.OAuthToken.RefreshToken,
				ExpiresAt:    a.config.OAuthToken.ExpiresAt,
			}, model)
		}
	case "claude-oauth":
		if a.config != nil && a.config.ClaudeOAuthToken != nil {
			a.provider = NewClaudeOAuthProvider(&OAuthToken{
				AccessToken:  a.config.ClaudeOAuthToken.AccessToken,
				RefreshToken: a.config.ClaudeOAuthToken.RefreshToken,
				ExpiresAt:    a.config.ClaudeOAuthToken.ExpiresAt,
			}, model)
		}
	default:
		return fmt.Errorf("알 수 없는 프로바이더: %s", providerType)
	}
	lang := "auto"
	if a.config != nil && a.config.Language != "" {
		lang = a.config.Language
	}
	a.config = &AppConfig{ProviderType: providerType, APIKey: apiKey, Model: model, Language: lang}
	return saveConfig(a.config)
}

// SetModel - 모델 변경
func (a *App) SetModel(model string) error {
	if a.config == nil {
		return fmt.Errorf("먼저 프로바이더를 설정하세요")
	}
	a.config.Model = model
	return a.SetProvider(a.config.ProviderType, a.config.APIKey)
}

// SetLanguage - 응답 언어 변경
func (a *App) SetLanguage(language string) error {
	if a.config == nil {
		a.config = &AppConfig{ProviderType: "claude-cli", Language: language}
	} else {
		a.config.Language = language
	}
	return saveConfig(a.config)
}

// ConnectChatGPTOAuth - ChatGPT 브라우저 OAuth 로그인
func (a *App) ConnectChatGPTOAuth() error {
	token, err := StartOpenAIOAuth(a.ctx)
	if err != nil {
		return err
	}
	model, lang := a.modelAndLang()
	a.provider = NewChatGPTOAuthProvider(token, model)
	a.config = &AppConfig{
		ProviderType: "chatgpt-oauth",
		Model:        model,
		Language:     lang,
		OAuthToken:   savedToken(token),
	}
	return saveConfig(a.config)
}

// ConnectClaudeOAuth - Claude.ai 브라우저 OAuth 로그인
func (a *App) ConnectClaudeOAuth() error {
	token, err := StartClaudeOAuth(a.ctx)
	if err != nil {
		return err
	}
	model, lang := a.modelAndLang()
	a.provider = NewClaudeOAuthProvider(token, model)
	a.config = &AppConfig{
		ProviderType:     "claude-oauth",
		Model:            model,
		Language:         lang,
		ClaudeOAuthToken: savedToken(token),
	}
	return saveConfig(a.config)
}

func (a *App) modelAndLang() (model, lang string) {
	if a.config != nil {
		model = a.config.Model
		lang = a.config.Language
	}
	if lang == "" {
		lang = "auto"
	}
	return
}

func savedToken(t *OAuthToken) *SavedToken {
	return &SavedToken{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresAt:    t.ExpiresAt,
	}
}

// StartSession - Step1 개념 선택 시 Claude CLI 세션 미리 시작 (워밍업)
func (a *App) StartSession(topic string) error {
	cli, ok := a.provider.(*ClaudeCLIProvider)
	if !ok {
		return nil
	}
	cli.sessionID = ""
	lang := "auto"
	if a.config != nil {
		lang = a.config.Language
	}
	initMsg := fmt.Sprintf(`We're about to practice the Feynman technique on the topic "%s". Please reply with just: "Ready."`, topic)
	_, err := cli.Analyze(a.ctx, "", initMsg, lang)
	return err
}

// AnalyzeExplanation - 동기 분석 (레거시)
func (a *App) AnalyzeExplanation(topic string, explanation string) (string, error) {
	lang := "auto"
	if a.config != nil {
		lang = a.config.Language
	}
	return a.provider.Analyze(a.ctx, topic, explanation, lang)
}

// GetSessionID - 현재 Claude CLI 세션 ID 반환
func (a *App) GetSessionID() string {
	if cli, ok := a.provider.(*ClaudeCLIProvider); ok {
		return cli.sessionID
	}
	return ""
}

// SendChatMessage - 채팅 메시지 전송 (동기)
func (a *App) SendChatMessage(history []ChatMessage, message string) (string, error) {
	return a.provider.Chat(a.ctx, history, message)
}

// AnalyzeStreaming - 스트리밍 분석 (이벤트: stream:chunk, stream:done, stream:error)
func (a *App) AnalyzeStreaming(topic string, explanation string) {
	go func() {
		lang := "auto"
		if a.config != nil {
			lang = a.config.Language
		}
		emit := func(chunk string) {
			wailsruntime.EventsEmit(a.ctx, "stream:chunk", chunk)
		}
		// ClaudeCLI는 새 분석마다 세션 초기화
		if cli, ok := a.provider.(*ClaudeCLIProvider); ok {
			cli.sessionID = ""
		}
		var err error
		if sp, ok := a.provider.(StreamingProvider); ok {
			err = sp.Stream(a.ctx, buildPrompt(topic, explanation, lang), emit)
		} else {
			var result string
			result, err = a.provider.Analyze(a.ctx, topic, explanation, lang)
			if err == nil {
				emit(result)
			}
		}
		if err != nil {
			wailsruntime.EventsEmit(a.ctx, "stream:error", err.Error())
			return
		}
		wailsruntime.EventsEmit(a.ctx, "stream:done", "")
	}()
}

// ChatStreaming - 채팅 스트리밍 (이벤트: stream:chunk, stream:done, stream:error)
func (a *App) ChatStreaming(history []ChatMessage, message string) {
	go func() {
		emit := func(chunk string) {
			wailsruntime.EventsEmit(a.ctx, "stream:chunk", chunk)
		}
		var err error
		if sp, ok := a.provider.(StreamingProvider); ok {
			err = sp.Stream(a.ctx, message, emit)
		} else {
			var result string
			result, err = a.provider.Chat(a.ctx, history, message)
			if err == nil {
				emit(result)
			}
		}
		if err != nil {
			wailsruntime.EventsEmit(a.ctx, "stream:error", err.Error())
			return
		}
		wailsruntime.EventsEmit(a.ctx, "stream:done", "")
	}()
}

// ContinueConversation - 세션 이어서 대화
func (a *App) ContinueConversation(sessionID string, message string) (string, error) {
	lang := "auto"
	if a.config != nil {
		lang = a.config.Language
	}
	if cli, ok := a.provider.(*ClaudeCLIProvider); ok {
		prev := cli.sessionID
		cli.sessionID = sessionID
		result, err := a.provider.Analyze(a.ctx, "", message, lang)
		cli.sessionID = prev
		return result, err
	}
	return a.provider.Analyze(a.ctx, "이전 대화 계속", message, lang)
}
