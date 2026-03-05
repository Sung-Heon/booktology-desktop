package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// AppConfig - 앱 설정 저장 구조체
type AppConfig struct {
	ProviderType string      `json:"provider_type"` // claude-cli, anthropic, openai, chatgpt-oauth
	APIKey       string      `json:"api_key,omitempty"`
	OAuthToken   *SavedToken `json:"oauth_token,omitempty"`
}

// SavedToken - 파일에 저장할 토큰 (OAuthToken과 분리)
type SavedToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "booktology", "config.json")
}

func loadConfig() (*AppConfig, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &AppConfig{ProviderType: "claude-cli"}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *AppConfig) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600) // 0600 = 본인만 읽기/쓰기 가능
}
