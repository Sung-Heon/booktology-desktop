package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicopt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openaiopt "github.com/openai/openai-go/option"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// 환경변수에서 특정 키 제거
func removeEnv(env []string, keys ...string) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		skip := false
		for _, key := range keys {
			if len(e) >= len(key)+1 && e[:len(key)+1] == key+"=" {
				skip = true
				break
			}
		}
		if !skip {
			result = append(result, e)
		}
	}
	return result
}

// stream-json 출력에서 텍스트 추출
func extractTextFromStreamJSON(data []byte) string {
	type resultMsg struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		Result  string `json:"result"`
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg resultMsg
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.Type == "result" && msg.Subtype == "success" && msg.Result != "" {
			return msg.Result
		}
	}
	// fallback: 원본 반환
	return string(data)
}

// ChatMessage - 대화 히스토리 항목
type ChatMessage struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

// AIProvider 인터페이스
type AIProvider interface {
	Analyze(ctx context.Context, topic string, explanation string, language string) (string, error)
	Chat(ctx context.Context, history []ChatMessage, message string) (string, error)
}

// ─── Claude CLI 프로바이더 ───────────────────────────────
type ClaudeCLIProvider struct {
	model     string
	sessionID string
}

func (p *ClaudeCLIProvider) Analyze(ctx context.Context, topic string, explanation string, language string) (string, error) {
	prompt := explanation
	if topic != "" {
		prompt = buildPrompt(topic, explanation, language)
	}
	args := []string{"-p", prompt, "--output-format", "stream-json", "--verbose"}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	if p.sessionID != "" {
		args = append(args, "--resume", p.sessionID)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Env = removeEnv(os.Environ(), "CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT")

	t0 := time.Now()
	fmt.Printf("[Claude CLI] 시작: %s\n", t0.Format("15:04:05.000"))
	fmt.Printf("[Claude CLI] 명령어: claude %v\n", args[:2]) // 프롬프트 제외

	output, err := cmd.CombinedOutput()
	elapsed := time.Since(t0)
	fmt.Printf("[Claude CLI] 완료: %.2fs\n", elapsed.Seconds())

	if err != nil {
		return "", fmt.Errorf("claude CLI 실행 실패: %w\n출력: %s", err, string(output))
	}

	if sid := p.extractSessionID(output); sid != "" {
		p.sessionID = sid
		fmt.Printf("[Claude CLI] 세션ID: %s\n", sid)
	}
	return extractTextFromStreamJSON(output), nil
}

func (p *ClaudeCLIProvider) Chat(ctx context.Context, _ []ChatMessage, message string) (string, error) {
	return p.Analyze(ctx, "", message, "")
}

func (p *ClaudeCLIProvider) streamAnalyze(ctx context.Context, topic string, explanation string, language string, emit func(string)) error {
	prompt := explanation
	if topic != "" {
		prompt = buildPrompt(topic, explanation, language)
	}
	args := []string{"-p", prompt, "--output-format", "stream-json", "--verbose"}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	if p.sessionID != "" {
		args = append(args, "--resume", p.sessionID)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Env = removeEnv(os.Environ(), "CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe 실패: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("claude CLI 시작 실패: %w", err)
	}

	type contentDelta struct {
		Type  string `json:"type"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}
	type initMsg struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}

	type resultMsg struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		Result  string `json:"result"`
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	emitted := false
	for scanner.Scan() {
		line := scanner.Bytes()
		var im initMsg
		if json.Unmarshal(line, &im) == nil && im.SessionID != "" {
			p.sessionID = im.SessionID
		}
		var cd contentDelta
		if json.Unmarshal(line, &cd) == nil && cd.Type == "content_block_delta" && cd.Delta.Type == "text_delta" && cd.Delta.Text != "" {
			emit(cd.Delta.Text)
			emitted = true
		}
		// 스트리밍 미지원 시 result 폴백
		if !emitted {
			var rm resultMsg
			if json.Unmarshal(line, &rm) == nil && rm.Type == "result" && rm.Subtype == "success" && rm.Result != "" {
				emit(rm.Result)
				emitted = true
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude CLI 실패: %w\n%s", err, stderr.String())
	}
	return nil
}

func (p *ClaudeCLIProvider) extractSessionID(data []byte) string {
	type initMsg struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var msg initMsg
		if json.Unmarshal(scanner.Bytes(), &msg) == nil && msg.SessionID != "" {
			return msg.SessionID
		}
	}
	return ""
}

// ─── Codex CLI 프로바이더 ────────────────────────────────
// oh-my-claudecode와 동일한 방식: codex exec -m {model} --json --full-auto
type CodexCLIProvider struct {
	model string
}

func (p *CodexCLIProvider) Analyze(ctx context.Context, topic string, explanation string, language string) (string, error) {
	model := p.model
	if model == "" {
		model = "gpt-5.3-codex"
	}
	prompt := buildPrompt(topic, explanation, language)
	cmd := exec.CommandContext(ctx, "codex", "exec", "-m", model,
		"--json", "--full-auto", "--skip-git-repo-check", "--", prompt)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("codex CLI 실행 실패: %w", err)
	}
	return string(output), nil
}

func (p *CodexCLIProvider) Chat(ctx context.Context, _ []ChatMessage, message string) (string, error) {
	return p.Analyze(ctx, "이전 대화 계속", message, "")
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

// ─── OpenAI API Key 프로바이더 ───────────────────────────
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
	token  *OAuthToken
	client *openai.Client
	model  string
}

func NewChatGPTOAuthProvider(token *OAuthToken, model string) *ChatGPTOAuthProvider {
	client := openai.NewClient(
		openaiopt.WithAPIKey(token.AccessToken),
		openaiopt.WithBaseURL("https://api.openai.com/v1/"),
		openaiopt.WithHeader("Authorization", "Bearer "+token.AccessToken),
	)
	if model == "" {
		model = "gpt-4o"
	}
	return &ChatGPTOAuthProvider{token: token, client: &client, model: model}
}

func (p *ChatGPTOAuthProvider) Chat(ctx context.Context, history []ChatMessage, message string) (string, error) {
	if p.token.IsExpired() {
		newToken, err := refreshOAuthToken(ctx, p.token.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("토큰 갱신 실패: %w", err)
		}
		p.token = newToken
		client := openai.NewClient(openaiopt.WithAPIKey(newToken.AccessToken))
		p.client = &client
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

func (p *ChatGPTOAuthProvider) Analyze(ctx context.Context, topic string, explanation string, language string) (string, error) {
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

// ─── 공통 프롬프트 ───────────────────────────────────────
func buildPrompt(topic string, explanation string, language string) string {
	langInstruction := map[string]string{
		"auto": "Respond in the same language as the student's explanation above.",
		"ko":   "한국어로 답변해주세요.",
		"en":   "Please respond in English.",
		"ja":   "日本語で答えてください。",
		"zh":   "请用中文回答。",
	}
	lang, ok := langInstruction[language]
	if !ok {
		lang = langInstruction["auto"]
	}
	return fmt.Sprintf(`A student explained the concept of "%s" as follows:

---
%s
---

As a Feynman learning method tutor, please analyze:
1. What they understood well
2. Gaps or misconceptions in understanding
3. 2-3 key questions to deepen understanding

%s`, topic, explanation, lang)
}

// ─── App ─────────────────────────────────────────────────
type App struct {
	ctx           context.Context
	provider      AIProvider
	config        *AppConfig
	lastSessionID string // Claude CLI 세션 ID 저장
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
			token := &OAuthToken{
				AccessToken:  cfg.OAuthToken.AccessToken,
				RefreshToken: cfg.OAuthToken.RefreshToken,
				ExpiresAt:    cfg.OAuthToken.ExpiresAt,
			}
			a.provider = NewChatGPTOAuthProvider(token, cfg.Model)
		}
	}
}

// GetConfig - 현재 설정 반환 (React UI용)
func (a *App) GetConfig() AppConfig {
	if a.config == nil {
		return AppConfig{ProviderType: "claude-cli", Language: "auto"}
	}
	// API 키와 토큰은 보내지 않음 (보안)
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
		// OAuth는 ConnectChatGPTOAuth로만 설정, 여기선 기존 토큰 유지
		if a.config != nil && a.config.OAuthToken != nil {
			token := &OAuthToken{
				AccessToken:  a.config.OAuthToken.AccessToken,
				RefreshToken: a.config.OAuthToken.RefreshToken,
				ExpiresAt:    a.config.OAuthToken.ExpiresAt,
			}
			a.provider = NewChatGPTOAuthProvider(token, model)
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
	// 프로바이더 재생성
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

// ConnectChatGPTOAuth - 브라우저 OAuth 시작
func (a *App) ConnectChatGPTOAuth() error {
	token, err := StartOpenAIOAuth(a.ctx)
	if err != nil {
		return err
	}
	model := ""
	lang := "auto"
	if a.config != nil {
		model = a.config.Model
		lang = a.config.Language
	}
	a.provider = NewChatGPTOAuthProvider(token, model)
	a.config = &AppConfig{
		ProviderType: "chatgpt-oauth",
		Model:        model,
		Language:     lang,
		OAuthToken: &SavedToken{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresAt:    token.ExpiresAt,
		},
	}
	return saveConfig(a.config)
}

// StartSession - Step1 개념 선택 시 Claude CLI 세션 미리 시작 (워밍업)
func (a *App) StartSession(topic string) error {
	cli, ok := a.provider.(*ClaudeCLIProvider)
	if !ok {
		return nil // Claude CLI가 아니면 불필요
	}
	cli.sessionID = "" // 이전 세션 초기화
	lang := "auto"
	if a.config != nil {
		lang = a.config.Language
	}
	initMsg := fmt.Sprintf(`We're about to practice the Feynman technique on the topic "%s". Please reply with just: "Ready."`, topic)
	_, err := cli.Analyze(a.ctx, "", initMsg, lang)
	return err
}

// AnalyzeExplanation - React Step3에서 호출 (StartSession으로 워밍업된 세션 재사용)
func (a *App) AnalyzeExplanation(topic string, explanation string) (string, error) {
	lang := "auto"
	if a.config != nil {
		lang = a.config.Language
	}
	// StartSession이 호출되지 않은 경우를 대비해 세션 초기화
	if cli, ok := a.provider.(*ClaudeCLIProvider); ok && cli.sessionID == "" {
		// 세션 없으면 새로 시작 (워밍업 안 된 경우)
		_ = cli
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

// SendChatMessage - 채팅 히스토리와 함께 메시지 전송 (React 채팅 UI용)
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
		if cli, ok := a.provider.(*ClaudeCLIProvider); ok {
			cli.sessionID = "" // 항상 새 세션으로 분석 시작
			if err := cli.streamAnalyze(a.ctx, topic, explanation, lang, emit); err != nil {
				wailsruntime.EventsEmit(a.ctx, "stream:error", err.Error())
				return
			}
		} else {
			result, err := a.provider.Analyze(a.ctx, topic, explanation, lang)
			if err != nil {
				wailsruntime.EventsEmit(a.ctx, "stream:error", err.Error())
				return
			}
			emit(result)
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
		if cli, ok := a.provider.(*ClaudeCLIProvider); ok {
			if err := cli.streamAnalyze(a.ctx, "", message, "", emit); err != nil {
				wailsruntime.EventsEmit(a.ctx, "stream:error", err.Error())
				return
			}
		} else {
			result, err := a.provider.Chat(a.ctx, history, message)
			if err != nil {
				wailsruntime.EventsEmit(a.ctx, "stream:error", err.Error())
				return
			}
			emit(result)
		}
		wailsruntime.EventsEmit(a.ctx, "stream:done", "")
	}()
}

// ContinueConversation - Step3에서 추가 질문 시 세션 이어서 호출
func (a *App) ContinueConversation(sessionID string, message string) (string, error) {
	lang := "auto"
	if a.config != nil {
		lang = a.config.Language
	}
	// Claude CLI인 경우 세션 재사용
	if cli, ok := a.provider.(*ClaudeCLIProvider); ok {
		prev := cli.sessionID
		cli.sessionID = sessionID
		result, err := a.provider.Analyze(a.ctx, "", message, lang)
		cli.sessionID = prev
		return result, err
	}
	// 다른 프로바이더는 그냥 새 호출
	return a.provider.Analyze(a.ctx, "이전 대화 계속", message, lang)
}
