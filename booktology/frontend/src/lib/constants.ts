import type { ProviderType } from '../types'

export const MODELS: Record<ProviderType, { value: string; label: string }[]> = {
    'claude-cli': [
        { value: '', label: '기본값' },
        { value: 'claude-opus-4-6', label: 'Claude Opus 4.6' },
        { value: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6' },
        { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5' },
    ],
    'claude-persistent': [
        { value: '', label: '기본값' },
        { value: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6' },
        { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5' },
    ],
    'claude-oauth': [
        { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5 (빠름)' },
        { value: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6' },
        { value: 'claude-opus-4-6', label: 'Claude Opus 4.6 (강력)' },
    ],
    'anthropic': [
        { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5 (빠름)' },
        { value: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6' },
        { value: 'claude-opus-4-6', label: 'Claude Opus 4.6 (강력)' },
    ],
    'openai': [
        { value: 'gpt-4o-mini', label: 'GPT-4o Mini (빠름)' },
        { value: 'gpt-4o', label: 'GPT-4o' },
        { value: 'gpt-4-turbo', label: 'GPT-4 Turbo' },
    ],
    'chatgpt-oauth': [
        { value: 'gpt-4o-mini', label: 'GPT-4o Mini (빠름)' },
        { value: 'gpt-4o', label: 'GPT-4o' },
    ],
}

export const LANGUAGES = [
    { value: 'auto', label: '자동 감지 (설명 언어로 응답)' },
    { value: 'ko', label: '한국어' },
    { value: 'en', label: 'English' },
    { value: 'ja', label: '日本語' },
    { value: 'zh', label: '中文' },
]

export const STEPS = [
    { num: 1, label: '개념 선택' },
    { num: 2, label: '자유 설명' },
    { num: 3, label: 'Claude 분석' },
    { num: 4, label: '복습 & 정리' },
]

export const PROSE = `prose prose-invert prose-sm max-w-none
    prose-headings:text-white prose-headings:font-bold
    prose-h1:text-xl prose-h2:text-lg prose-h3:text-base prose-h3:text-indigo-300
    prose-p:text-gray-300 prose-p:leading-relaxed prose-strong:text-white
    prose-ul:text-gray-300 prose-ol:text-gray-300 prose-li:text-gray-300
    prose-blockquote:border-indigo-500 prose-blockquote:text-gray-400
    prose-code:text-green-400 prose-code:bg-gray-900 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded
    prose-pre:bg-gray-900 prose-pre:text-green-400 prose-hr:border-gray-700`
