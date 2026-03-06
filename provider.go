package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

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

// StreamingProvider - 스트리밍 지원 프로바이더 (선택적 확장)
type StreamingProvider interface {
	AIProvider
	Stream(ctx context.Context, prompt string, emit func(string)) error
}

// buildPrompt - 파인만 튜터 프롬프트 생성
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

// removeEnv - 환경변수에서 특정 키 제거
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

// extractTextFromStreamJSON - stream-json 출력에서 텍스트 추출
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
	return string(data)
}
