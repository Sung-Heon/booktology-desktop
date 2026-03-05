package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/browser"
)

const (
	openAIClientID    = "app_EMoamEEZ73f0CkXaXp7hrann"
	openAIAuthURL     = "https://auth.openai.com/oauth/authorize"
	openAITokenURL    = "https://auth.openai.com/oauth/token"
	openAIRedirectURI = "http://localhost:1455/auth/callback"
	openAIScopes      = "openid profile email offline_access"
)

// OAuthToken - 발급받은 토큰 저장
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (t *OAuthToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-5 * time.Minute))
}

// PKCE 생성
func generatePKCE() (verifier string, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return
}

// StartOpenAIOAuth - 브라우저 열고 OAuth 시작, 토큰 반환
func StartOpenAIOAuth(ctx context.Context) (*OAuthToken, error) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, fmt.Errorf("PKCE 생성 실패: %w", err)
	}

	// state 생성 (CSRF 방지용 랜덤값)
	stateBytes := make([]byte, 16)
	if _, err = rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("state 생성 실패: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// 인증 URL 생성
	params := url.Values{
		"response_type":              {"code"},
		"client_id":                  {openAIClientID},
		"redirect_uri":               {openAIRedirectURI},
		"scope":                      {openAIScopes},
		"code_challenge":             {challenge},
		"code_challenge_method":      {"S256"},
		"state":                      {state},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
	}
	authURL := openAIAuthURL + "?" + params.Encode()

	// 콜백 대기 채널
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// localhost:1455 콜백 서버 시작
	mux := http.NewServeMux()
	server := &http.Server{Addr: ":1455", Handler: mux}
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// state 검증
		if q.Get("state") != state {
			errCh <- fmt.Errorf("state 불일치 (CSRF 의심)")
			http.Error(w, "인증 실패", http.StatusBadRequest)
			return
		}
		// OpenAI가 에러 반환한 경우
		if errMsg := q.Get("error"); errMsg != "" {
			desc := q.Get("error_description")
			errCh <- fmt.Errorf("OAuth 오류: %s - %s", errMsg, desc)
			fmt.Fprintf(w, "<html><body><h2>인증 실패: %s</h2></body></html>", errMsg)
			return
		}
		code := q.Get("code")
		if code == "" {
			errCh <- fmt.Errorf("인증 코드가 없습니다 (전체 URL: %s)", r.URL.String())
			http.Error(w, "인증 실패", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, "<html><body><h2>인증 완료! 앱으로 돌아가세요.</h2></body></html>")
		codeCh <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Shutdown(context.Background())

	// 브라우저 열기
	if err := browser.OpenURL(authURL); err != nil {
		return nil, fmt.Errorf("브라우저 열기 실패: %w", err)
	}

	// 콜백 대기 (3분 타임아웃)
	select {
	case code := <-codeCh:
		return exchangeCodeForToken(ctx, code, verifier)
	case err := <-errCh:
		return nil, err
	case <-time.After(3 * time.Minute):
		return nil, fmt.Errorf("인증 시간 초과")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// 인증 코드 → 토큰 교환
func exchangeCodeForToken(ctx context.Context, code string, verifier string) (*OAuthToken, error) {
	body := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {openAIClientID},
		"code":          {code},
		"redirect_uri":  {openAIRedirectURI},
		"code_verifier": {verifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openAITokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("토큰 교환 실패: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("토큰 오류: %s", result.Error)
	}

	return &OAuthToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}

// 토큰 갱신
func refreshOAuthToken(ctx context.Context, refreshToken string) (*OAuthToken, error) {
	body := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {openAIClientID},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openAITokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("토큰 갱신 실패: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("갱신 오류: %s", result.Error)
	}

	return &OAuthToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}
