package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

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

	output, err := cmd.CombinedOutput()
	fmt.Printf("[Claude CLI] 완료: %.2fs\n", time.Since(t0).Seconds())

	if err != nil {
		return "", fmt.Errorf("claude CLI 실행 실패: %w\n출력: %s", err, string(output))
	}

	if sid := p.extractSessionID(output); sid != "" {
		p.sessionID = sid
	}
	return extractTextFromStreamJSON(output), nil
}

func (p *ClaudeCLIProvider) Chat(ctx context.Context, _ []ChatMessage, message string) (string, error) {
	return p.Analyze(ctx, "", message, "")
}

func (p *ClaudeCLIProvider) Stream(ctx context.Context, prompt string, emit func(string)) error {
	return p.streamAnalyze(ctx, "", prompt, "", emit)
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

// ─── Claude Persistent 프로바이더 (프로세스 유지) ──────────
type ClaudePersistentProvider struct {
	model   string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	mu      sync.Mutex
	env     []string
}

func NewClaudePersistentProvider(model string) *ClaudePersistentProvider {
	return &ClaudePersistentProvider{model: model}
}

func (p *ClaudePersistentProvider) start() error {
	args := []string{"--output-format", "stream-json", "--verbose"}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	p.cmd = exec.Command("claude", args...)
	p.cmd.Env = p.env

	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	p.cmd.Stderr = os.Stderr
	p.stdin = stdin
	p.scanner = bufio.NewScanner(stdout)
	p.scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	return p.cmd.Start()
}

func (p *ClaudePersistentProvider) isAlive() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	return p.cmd.ProcessState == nil
}

func (p *ClaudePersistentProvider) ensureAlive() error {
	if !p.isAlive() {
		return p.start()
	}
	return nil
}

func (p *ClaudePersistentProvider) ask(prompt string, emit func(string)) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ensureAlive(); err != nil {
		return fmt.Errorf("claude 프로세스 시작 실패: %w", err)
	}

	if _, err := fmt.Fprintln(p.stdin, prompt); err != nil {
		return fmt.Errorf("stdin 쓰기 실패: %w", err)
	}

	type contentDelta struct {
		Type  string `json:"type"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}
	type resultMsg struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		Result  string `json:"result"`
	}

	emitted := false
	for p.scanner.Scan() {
		line := p.scanner.Bytes()
		var cd contentDelta
		if json.Unmarshal(line, &cd) == nil && cd.Type == "content_block_delta" && cd.Delta.Type == "text_delta" && cd.Delta.Text != "" {
			emit(cd.Delta.Text)
			emitted = true
		}
		var rm resultMsg
		if json.Unmarshal(line, &rm) == nil && rm.Type == "result" {
			if rm.Subtype != "success" {
				return fmt.Errorf("claude 오류: %s", rm.Result)
			}
			if !emitted && rm.Result != "" {
				emit(rm.Result)
			}
			return nil
		}
	}
	return p.scanner.Err()
}

func (p *ClaudePersistentProvider) Analyze(ctx context.Context, topic string, explanation string, language string) (string, error) {
	prompt := explanation
	if topic != "" {
		prompt = buildPrompt(topic, explanation, language)
	}
	var buf strings.Builder
	err := p.ask(prompt, func(chunk string) { buf.WriteString(chunk) })
	return buf.String(), err
}

func (p *ClaudePersistentProvider) Chat(ctx context.Context, _ []ChatMessage, message string) (string, error) {
	var buf strings.Builder
	err := p.ask(message, func(chunk string) { buf.WriteString(chunk) })
	return buf.String(), err
}

func (p *ClaudePersistentProvider) Stream(ctx context.Context, prompt string, emit func(string)) error {
	return p.ask(prompt, emit)
}

func (p *ClaudePersistentProvider) Close() {
	if p.stdin != nil {
		p.stdin.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
}

// ─── Codex CLI 프로바이더 ────────────────────────────────
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
