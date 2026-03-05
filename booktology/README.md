# Booktology

파인만 학습법 기반 AI 튜터 데스크탑 앱 (macOS)

파인만 학습법(Feynman Technique)으로 개념을 깊이 이해하는 4단계 학습 사이클을 AI와 함께 진행합니다.

## 파인만 학습법이란?

> "내가 설명할 수 없다면, 아직 이해하지 못한 것이다" — Richard Feynman

1. **개념 선택** - 공부할 주제를 정한다
2. **자유롭게 설명** - 아무것도 보지 않고 내 말로 설명한다
3. **갭 분석** - AI가 이해 공백과 오개념을 짚어준다
4. **복습 및 단순화** - 다시 공부하고 더 쉽게 설명해본다

## 기능

- **멀티 AI 프로바이더** - Claude CLI, Anthropic API, OpenAI API, Codex CLI 지원
- **학습 세션 저장** - 사이드바에서 이전 학습 기록 확인
- **마크다운 렌더링** - AI 분석 결과를 깔끔하게 표시
- **다국어 응답** - 한국어, 영어, 일본어, 중국어, 자동 감지 선택 가능
- **대화 이어가기** - 분석 후 추가 질문으로 대화 지속
- **모델 선택** - 프로바이더별 원하는 모델 직접 지정

## 설치 및 실행

### 사전 요구사항

- macOS
- Go 1.21+
- Wails v2
- Node.js 18+
- Claude Code CLI (Claude CLI 프로바이더 사용 시)

```bash
# Go 설치
brew install go

# Wails 설치
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH=$PATH:$HOME/go/bin
```

### 개발 모드 실행

```bash
git clone https://github.com/Sung-Heon/booktology-desktop.git
cd booktology-desktop
wails dev
```

### 빌드

```bash
wails build
# 결과물: build/bin/booktology.app
```

## AI 프로바이더 설정

앱 우측 상단 설정 아이콘에서 변경 가능합니다.

| 프로바이더 | 설명 | 필요한 것 |
|-----------|------|----------|
| Claude CLI (기본) | 로컬 Claude Code CLI 실행 | Claude Code 설치 및 로그인 |
| Anthropic API | Anthropic API 직접 호출 | API 키 |
| OpenAI API | OpenAI API 직접 호출 | API 키 |
| Codex CLI | OpenAI Codex CLI 실행 | Codex CLI 설치 |

## 기술 스택

- **백엔드**: Go + Wails v2
- **프론트엔드**: React + TypeScript + Tailwind CSS v4
- **AI**: Claude Code CLI / Anthropic SDK / OpenAI SDK
- **마크다운**: react-markdown + remark-gfm

## 프로젝트 구조

```
booktology-desktop/
├── app.go          # AI 프로바이더 로직 (Claude CLI, API 등)
├── config.go       # 설정 저장/불러오기
├── oauth.go        # OAuth 인증 흐름
├── main.go         # Wails 앱 진입점
├── build/          # 빌드 설정 및 아이콘
├── frontend/
│   ├── src/
│   │   ├── App.tsx     # 메인 React 컴포넌트
│   │   └── style.css   # Tailwind CSS
│   └── wailsjs/        # Wails JS 바인딩 (자동 생성)
└── wails.json      # Wails 프로젝트 설정
```

## 설정 파일 위치

앱 설정은 `~/.config/booktology/config.json`에 저장됩니다.

## 라이선스

MIT
