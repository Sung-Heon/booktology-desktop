import { useState, useEffect } from 'react'
import {
    SetProvider, ConnectChatGPTOAuth, ConnectClaudeOAuth,
    GetConfig, SetModel, SetLanguage,
} from '../../wailsjs/go/main/App'
import { MODELS, LANGUAGES } from '../lib/constants'
import type { ProviderType } from '../types'

const PROVIDERS: { id: ProviderType; label: string; desc: string; dim: boolean; oauth?: boolean }[] = [
    { id: 'claude-cli',        label: 'Claude Code CLI',    desc: '매 요청마다 새 프로세스 (안정적)',                  dim: false },
    { id: 'claude-persistent', label: 'Claude Persistent',  desc: '프로세스 유지 — 빠른 대화 (실험적)',              dim: false },
    { id: 'claude-oauth',      label: 'Claude OAuth',       desc: 'Claude.ai 계정으로 브라우저 로그인 (API 키 불필요)', dim: false, oauth: true },
    { id: 'anthropic',         label: 'Anthropic API',      desc: 'Claude API 직접 호출 (빠름, API 키 필요)',        dim: true },
    { id: 'openai',            label: 'OpenAI API 키',      desc: 'ChatGPT API 키로 연결',                         dim: true },
    { id: 'chatgpt-oauth',     label: 'ChatGPT OAuth',      desc: 'ChatGPT Plus/Pro 구독으로 브라우저 로그인',       dim: true, oauth: true },
]

interface Props { onBack: () => void }

export function Settings({ onBack }: Props) {
    const [provider, setProvider] = useState<ProviderType>('claude-cli')
    const [activeProvider, setActiveProvider] = useState<ProviderType>('claude-cli')
    const [apiKey, setApiKey] = useState('')
    const [model, setModel] = useState('')
    const [language, setLanguage] = useState('auto')
    const [saved, setSaved] = useState(false)
    const [error, setError] = useState('')
    const [connecting, setConnecting] = useState(false)

    useEffect(() => {
        GetConfig().then(cfg => {
            setActiveProvider(cfg.provider_type as ProviderType)
            setProvider(cfg.provider_type as ProviderType)
            setModel(cfg.model || '')
            setLanguage(cfg.language || 'auto')
        })
    }, [])

    function save() {
        SetProvider(provider, apiKey)
            .then(() => SetModel(model))
            .then(() => SetLanguage(language))
            .then(() => { setSaved(true); setError(''); setActiveProvider(provider) })
            .catch(err => setError(String(err)))
    }

    function connectOAuth(fn: () => Promise<void>, newProvider: ProviderType) {
        setConnecting(true)
        setError('')
        fn()
            .then(() => SetModel(model))
            .then(() => SetLanguage(language))
            .then(() => { setSaved(true); setActiveProvider(newProvider) })
            .catch(err => setError(String(err)))
            .finally(() => setConnecting(false))
    }

    const selectedInfo = PROVIDERS.find(p => p.id === provider)
    const isOAuth = selectedInfo?.oauth && !selectedInfo.dim

    return (
        <div className="max-w-xl">
            <h2 className="text-2xl font-bold mb-2">AI 프로바이더 설정</h2>
            <p className="text-gray-400 mb-6">사용할 AI를 선택하고 설정하세요.</p>

            <div className="space-y-3 mb-6">
                {PROVIDERS.map(p => (
                    <div
                        key={p.id}
                        onClick={() => { if (!p.dim) { setProvider(p.id); setApiKey(''); setSaved(false) } }}
                        className={`p-4 rounded-xl border transition-colors
                            ${p.dim ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'}
                            ${!p.dim && provider === p.id ? 'border-indigo-500 bg-indigo-950' : 'border-gray-700 bg-gray-800'}
                            ${!p.dim && provider !== p.id ? 'hover:border-gray-600' : ''}`}
                    >
                        <div className="flex items-center gap-2">
                            <span className="font-medium">{p.label}</span>
                            {activeProvider === p.id && (
                                <span className="text-xs bg-green-600 text-white px-2 py-0.5 rounded-full">활성</span>
                            )}
                            {p.dim && <span className="text-xs text-gray-600">준비 중</span>}
                        </div>
                        <div className="text-sm text-gray-400 mt-1">{p.desc}</div>
                    </div>
                ))}
            </div>

            {(provider === 'anthropic' || provider === 'openai') && (
                <input
                    type="password"
                    value={apiKey}
                    onChange={e => { setApiKey(e.target.value); setSaved(false) }}
                    placeholder={provider === 'anthropic' ? 'sk-ant-...' : 'sk-...'}
                    className="w-full bg-gray-800 rounded-lg px-4 py-3 text-white placeholder-gray-500 outline-none focus:ring-2 focus:ring-indigo-500 mb-4 font-mono text-sm"
                />
            )}

            <div className="mb-4">
                <label className="block text-sm text-gray-400 mb-2">모델</label>
                <select
                    value={model}
                    onChange={e => { setModel(e.target.value); setSaved(false) }}
                    className="w-full bg-gray-800 rounded-lg px-4 py-3 text-white outline-none focus:ring-2 focus:ring-indigo-500"
                >
                    {(MODELS[provider] || []).map(m => (
                        <option key={m.value} value={m.value}>{m.label}</option>
                    ))}
                </select>
            </div>

            <div className="mb-6">
                <label className="block text-sm text-gray-400 mb-2">응답 언어</label>
                <select
                    value={language}
                    onChange={e => { setLanguage(e.target.value); setSaved(false) }}
                    className="w-full bg-gray-800 rounded-lg px-4 py-3 text-white outline-none focus:ring-2 focus:ring-indigo-500"
                >
                    {LANGUAGES.map(l => (
                        <option key={l.value} value={l.value}>{l.label}</option>
                    ))}
                </select>
            </div>

            {error && <p className="text-red-400 text-sm mb-3">{error}</p>}
            {saved && <p className="text-green-400 text-sm mb-3">저장됐어요!</p>}

            <div className="flex gap-3">
                {isOAuth ? (
                    <button
                        onClick={() => connectOAuth(
                            provider === 'claude-oauth' ? ConnectClaudeOAuth : ConnectChatGPTOAuth,
                            provider
                        )}
                        disabled={connecting}
                        className={`flex-1 disabled:bg-gray-700 disabled:text-gray-500 rounded-lg py-3 font-semibold transition-colors
                            ${provider === 'claude-oauth'
                                ? 'bg-orange-600 hover:bg-orange-700'
                                : 'bg-green-600 hover:bg-green-700'}`}
                    >
                        {connecting ? '브라우저에서 로그인 중...' : provider === 'claude-oauth' ? 'Claude.ai 브라우저 로그인' : 'ChatGPT 브라우저 로그인'}
                    </button>
                ) : (
                    <button onClick={save} className="flex-1 bg-indigo-600 hover:bg-indigo-700 rounded-lg py-3 font-semibold transition-colors">
                        저장
                    </button>
                )}
                <button onClick={onBack} className="flex-1 bg-gray-700 hover:bg-gray-600 rounded-lg py-3 font-semibold transition-colors">
                    뒤로
                </button>
            </div>
        </div>
    )
}
