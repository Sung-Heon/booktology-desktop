export type ChatMsg = {
    role: 'user' | 'assistant'
    content: string
    streaming?: boolean
}

export interface SessionRecord {
    id: string
    topic: string
    explanation: string
    analysis: string
    score: number
    createdAt: string
}

export type ProviderType =
    | 'claude-cli'
    | 'claude-persistent'
    | 'claude-oauth'
    | 'anthropic'
    | 'openai'
    | 'chatgpt-oauth'

export type Page = 'learn' | 'settings' | 'history'
