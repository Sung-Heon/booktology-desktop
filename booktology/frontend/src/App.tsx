import { useState, useEffect, useRef } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { AnalyzeStreaming, ChatStreaming, StartSession, SetProvider, ConnectChatGPTOAuth, GetConfig, SetModel, SetLanguage } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

const MODELS: Record<string, { value: string; label: string }[]> = {
    'claude-cli': [
        { value: '', label: '기본값' },
        { value: 'claude-opus-4-6', label: 'Claude Opus 4.6' },
        { value: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6' },
        { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5' },
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

const LANGUAGES = [
    { value: 'auto', label: '자동 감지 (설명 언어로 응답)' },
    { value: 'ko', label: '한국어' },
    { value: 'en', label: 'English' },
    { value: 'ja', label: '日本語' },
    { value: 'zh', label: '中文' },
]

const STEPS = [
    { num: 1, label: '개념 선택' },
    { num: 2, label: '자유 설명' },
    { num: 3, label: 'Claude 분석' },
    { num: 4, label: '복습 & 정리' },
]

function StepIndicator({ current }: { current: number }) {
    return (
        <div className="flex items-center gap-2 mb-8">
            {STEPS.map((s, i) => (
                <div key={s.num} className="flex items-center gap-2">
                    <div className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium
                        ${current === s.num ? 'bg-indigo-600 text-white' : current > s.num ? 'bg-indigo-900 text-indigo-300' : 'bg-gray-800 text-gray-500'}`}>
                        <span>{s.num}</span>
                        <span>{s.label}</span>
                    </div>
                    {i < STEPS.length - 1 && <div className="w-6 h-px bg-gray-700" />}
                </div>
            ))}
        </div>
    )
}

function Step1({ onNext }: { onNext: (topic: string) => void }) {
    const [topic, setTopic] = useState('')
    return (
        <div className="max-w-xl">
            <h2 className="text-2xl font-bold mb-2">어떤 개념을 공부할까요?</h2>
            <p className="text-gray-400 mb-6">학습하고 싶은 개념을 입력하세요. Claude가 이해도를 분석해줄 거예요.</p>
            <input
                type="text"
                value={topic}
                onChange={e => setTopic(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && topic.trim() && onNext(topic)}
                placeholder="예) 양자역학, 이진탐색, 광합성..."
                className="w-full bg-gray-800 rounded-lg px-4 py-3 text-white placeholder-gray-500 outline-none focus:ring-2 focus:ring-indigo-500 mb-4"
            />
            <button
                onClick={() => topic.trim() && onNext(topic)}
                className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-700 disabled:text-gray-500 rounded-lg py-3 font-semibold transition-colors"
                disabled={!topic.trim()}
            >
                학습 시작 →
            </button>
        </div>
    )
}

function Step2({ topic, onNext, sessionReady }: { topic: string; onNext: (text: string) => void; sessionReady: boolean | null }) {
    const [text, setText] = useState('')
    return (
        <div className="max-w-2xl w-full">
            <h2 className="text-2xl font-bold mb-2">"{topic}"을 설명해보세요</h2>
            <p className="text-gray-400 mb-4">초등학생에게 설명한다고 생각하고 자유롭게 써보세요. 틀려도 괜찮아요.</p>
            {sessionReady === false && (
                <div className="flex items-center gap-2 text-xs text-gray-500 mb-3">
                    <div className="w-2.5 h-2.5 rounded-full border border-indigo-500 border-t-transparent animate-spin" />
                    Claude CLI 세션 준비 중...
                </div>
            )}
            {sessionReady === true && (
                <div className="flex items-center gap-2 text-xs text-green-500 mb-3">
                    <div className="w-2 h-2 rounded-full bg-green-500" />
                    세션 준비 완료 — 분석이 더 빠르게 시작돼요
                </div>
            )}
            <textarea
                value={text}
                onChange={e => setText(e.target.value)}
                placeholder="여기에 자유롭게 설명을 작성하세요..."
                rows={12}
                className="w-full bg-gray-800 rounded-lg px-4 py-3 text-white placeholder-gray-500 outline-none focus:ring-2 focus:ring-indigo-500 resize-none mb-4"
            />
            <button
                onClick={() => text.trim() && onNext(text)}
                disabled={!text.trim()}
                className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-700 disabled:text-gray-500 rounded-lg py-3 font-semibold transition-colors"
            >
                Claude에게 분석 요청 →
            </button>
        </div>
    )
}

type ChatMsg = { role: 'user' | 'assistant'; content: string; streaming?: boolean }

const PROSE = `prose prose-invert prose-sm max-w-none
    prose-headings:text-white prose-headings:font-bold
    prose-h1:text-xl prose-h2:text-lg prose-h3:text-base prose-h3:text-indigo-300
    prose-p:text-gray-300 prose-p:leading-relaxed prose-strong:text-white
    prose-ul:text-gray-300 prose-ol:text-gray-300 prose-li:text-gray-300
    prose-blockquote:border-indigo-500 prose-blockquote:text-gray-400
    prose-code:text-green-400 prose-code:bg-gray-900 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded
    prose-pre:bg-gray-900 prose-pre:text-green-400 prose-hr:border-gray-700`

function Step3({ topic, explanation, onAnalyzed, onNext }: { topic: string; explanation: string; onAnalyzed: (a: string) => void; onNext: () => void }) {
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')
    const [chatMessages, setChatMessages] = useState<ChatMsg[]>([])
    const [chatInput, setChatInput] = useState('')
    const [chatLoading, setChatLoading] = useState(false)
    const chatEndRef = useRef<HTMLDivElement>(null)
    const analysisRef = useRef('')

    useEffect(() => {
        // 유저 설명 + AI 스트리밍 자리 초기 설정
        setChatMessages([
            { role: 'user', content: explanation },
            { role: 'assistant', content: '', streaming: true },
        ])

        const offChunk = EventsOn('stream:chunk', (chunk: string) => {
            analysisRef.current += chunk
            setChatMessages(prev => {
                const last = prev[prev.length - 1]
                if (last?.streaming) return [...prev.slice(0, -1), { ...last, content: analysisRef.current }]
                return prev
            })
        })
        const offDone = EventsOn('stream:done', () => {
            setChatMessages(prev => prev.map((m, i) => i === prev.length - 1 ? { ...m, content: analysisRef.current, streaming: false } : m))
            onAnalyzed(analysisRef.current)
            setLoading(false)
        })
        const offError = EventsOn('stream:error', (err: string) => {
            setError(err)
            setLoading(false)
        })

        AnalyzeStreaming(topic, explanation)

        return () => { offChunk(); offDone(); offError() }
    }, [])

    useEffect(() => {
        chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, [chatMessages, chatLoading])

    function sendMessage() {
        const msg = chatInput.trim()
        if (!msg || chatLoading) return
        const newHistory = [...chatMessages, { role: 'user' as const, content: msg }]
        setChatMessages([...newHistory, { role: 'assistant' as const, content: '', streaming: true }])
        setChatInput('')
        setChatLoading(true)

        const offChunk = EventsOn('stream:chunk', (chunk: string) => {
            setChatMessages(prev => {
                const last = prev[prev.length - 1]
                if (last?.streaming) return [...prev.slice(0, -1), { ...last, content: last.content + chunk }]
                return prev
            })
        })
        const offDone = EventsOn('stream:done', () => {
            setChatMessages(prev => prev.map((m, i) => i === prev.length - 1 ? { ...m, streaming: false } : m))
            setChatLoading(false)
            offChunk(); offDone(); offError()
        })
        const offError = EventsOn('stream:error', (err: string) => {
            setChatMessages(prev => [...prev.slice(0, -1), { role: 'assistant' as const, content: `오류: ${err}`, streaming: false }])
            setChatLoading(false)
            offChunk(); offDone(); offError()
        })

        ChatStreaming(newHistory, msg)
    }

    return (
        <div className="max-w-2xl w-full flex flex-col" style={{ height: 'calc(100vh - 160px)' }}>
            <h2 className="text-2xl font-bold mb-1">AI 분석 & 대화</h2>
            <p className="text-gray-400 mb-3 text-sm">분석 결과를 보고 추가 질문을 이어갈 수 있어요.</p>

            {/* 대화 영역 */}
            <div className="flex-1 overflow-auto bg-gray-900 rounded-xl border border-gray-800 p-4 space-y-4 mb-3">
                {error && <p className="text-red-400 text-sm">오류: {error}</p>}
                {chatMessages.map((msg, i) => (
                    <div key={i} className={`flex gap-2 ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                        {msg.role === 'assistant' && (
                            <div className="w-7 h-7 rounded-full bg-indigo-700 flex items-center justify-center text-xs font-bold shrink-0 mt-1">AI</div>
                        )}
                        <div className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm
                            ${msg.role === 'user'
                                ? 'bg-indigo-600 text-white rounded-tr-sm'
                                : 'bg-gray-800 text-gray-200 rounded-tl-sm border border-gray-700'}`}>
                            {msg.role === 'assistant' ? (
                                msg.content ? (
                                    <>
                                        <div className={PROSE}>
                                            <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
                                        </div>
                                        {msg.streaming && (
                                            <span className="inline-block w-0.5 h-3.5 bg-indigo-400 animate-pulse ml-0.5 align-middle" />
                                        )}
                                    </>
                                ) : (
                                    <div className="flex items-center gap-2 text-gray-500 py-1">
                                        <div className="w-3 h-3 rounded-full border-2 border-indigo-400 border-t-transparent animate-spin" />
                                        <span>응답 생성 중...</span>
                                    </div>
                                )
                            ) : msg.content}
                        </div>
                        {msg.role === 'user' && (
                            <div className="w-7 h-7 rounded-full bg-gray-600 flex items-center justify-center text-xs shrink-0 mt-1">나</div>
                        )}
                    </div>
                ))}
                <div ref={chatEndRef} />
            </div>

            {/* 입력창 */}
            {!loading && !error && (
                <div className="flex gap-2 mb-3">
                    <input
                        type="text"
                        value={chatInput}
                        onChange={e => setChatInput(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && !e.shiftKey && sendMessage()}
                        placeholder="추가 질문을 입력하세요..."
                        disabled={chatLoading}
                        className="flex-1 bg-gray-800 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 outline-none focus:ring-2 focus:ring-indigo-500 text-sm disabled:opacity-50"
                    />
                    <button
                        onClick={sendMessage}
                        disabled={!chatInput.trim() || chatLoading}
                        className="bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-700 disabled:text-gray-500 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors"
                    >
                        전송
                    </button>
                </div>
            )}

            <button
                onClick={onNext}
                disabled={loading}
                className="w-full bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:text-gray-600 rounded-lg py-2.5 text-sm font-semibold transition-colors"
            >
                복습 단계로 →
            </button>
        </div>
    )
}

function Step4({ topic, onRestart, onSave }: { topic: string; onRestart: () => void; onSave: (score: number) => void }) {
    const [score, setScore] = useState(0)
    const [saved, setSaved] = useState(false)
    return (
        <div className="max-w-xl">
            <h2 className="text-2xl font-bold mb-2">학습 완료!</h2>
            <p className="text-gray-400 mb-6">"{topic}" 학습 세션을 마쳤어요.</p>
            <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 mb-6">
                <p className="text-sm text-gray-400 mb-4">이해도를 평가해주세요</p>
                <div className="flex gap-2">
                    {[1,2,3,4,5,6,7,8,9,10].map(n => (
                        <button
                            key={n}
                            onClick={() => setScore(n)}
                            className={`flex-1 py-2 rounded-lg text-sm font-bold transition-colors
                                ${score >= n ? 'bg-indigo-600 text-white' : 'bg-gray-700 text-gray-400 hover:bg-gray-600'}`}
                        >
                            {n}
                        </button>
                    ))}
                </div>
            </div>
            <div className="flex gap-3">
                <button
                    onClick={() => { if (!saved) { onSave(score); setSaved(true) } }}
                    disabled={score === 0 || saved}
                    className="flex-1 bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-700 disabled:text-gray-500 rounded-lg py-3 font-semibold transition-colors"
                >
                    {saved ? '저장됨 ✓' : '기록 저장'}
                </button>
                <button
                    onClick={onRestart}
                    className="flex-1 bg-gray-700 hover:bg-gray-600 rounded-lg py-3 font-semibold transition-colors"
                >
                    새 개념 학습하기
                </button>
            </div>
        </div>
    )
}

type ProviderType = 'claude-cli' | 'anthropic' | 'openai' | 'chatgpt-oauth'

function Settings({ onBack }: { onBack: () => void }) {
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

    function connectOAuth() {
        setConnecting(true)
        setError('')
        ConnectChatGPTOAuth()
            .then(() => SetModel(model))
            .then(() => SetLanguage(language))
            .then(() => { setSaved(true); setActiveProvider('chatgpt-oauth') })
            .catch(err => setError(String(err)))
            .finally(() => setConnecting(false))
    }

    return (
        <div className="max-w-xl">
            <h2 className="text-2xl font-bold mb-2">AI 프로바이더 설정</h2>
            <p className="text-gray-400 mb-6">사용할 AI를 선택하고 설정하세요.</p>

            <div className="space-y-3 mb-6">
                {([
                    { id: 'claude-cli', label: 'Claude Code CLI', desc: '로컬 claude CLI 사용 (API 키 불필요)', dim: false },
                    { id: 'anthropic', label: 'Anthropic API', desc: 'Claude API 직접 호출 (빠름, API 키 필요)', dim: true },
                    { id: 'openai', label: 'OpenAI API 키', desc: 'ChatGPT API 키로 연결', dim: true },
                    { id: 'chatgpt-oauth', label: 'ChatGPT OAuth', desc: 'ChatGPT Plus/Pro 구독으로 브라우저 로그인', dim: true },
                ] as const).map(p => (
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

            {/* 모델 선택 */}
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

            {/* 언어 선택 */}
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
                {provider === 'chatgpt-oauth' ? (
                    <button
                        onClick={connectOAuth}
                        disabled={connecting}
                        className="flex-1 bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 rounded-lg py-3 font-semibold transition-colors"
                    >
                        {connecting ? '브라우저에서 로그인 중...' : '브라우저로 로그인'}
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

type Page = 'learn' | 'settings' | 'history'

interface SessionRecord {
    id: string
    topic: string
    explanation: string
    analysis: string
    score: number
    createdAt: string
}

function loadSessions(): SessionRecord[] {
    try { return JSON.parse(localStorage.getItem('booktology-sessions') || '[]') }
    catch { return [] }
}

function saveSession(s: SessionRecord) {
    const sessions = loadSessions()
    localStorage.setItem('booktology-sessions', JSON.stringify([s, ...sessions].slice(0, 50)))
}

function HistoryView({ sessions, onSelect }: { sessions: SessionRecord[]; onSelect: (s: SessionRecord) => void }) {
    if (sessions.length === 0) return (
        <div className="text-gray-500 text-sm mt-8">아직 학습 기록이 없어요.</div>
    )
    return (
        <div className="max-w-2xl w-full">
            <h2 className="text-2xl font-bold mb-6">학습 기록</h2>
            <div className="space-y-3">
                {sessions.map(s => (
                    <div key={s.id} onClick={() => onSelect(s)}
                        className="bg-gray-900 rounded-xl p-4 border border-gray-800 hover:border-indigo-500 cursor-pointer transition-colors">
                        <div className="flex items-center justify-between mb-1">
                            <span className="font-semibold text-white">{s.topic}</span>
                            {s.score > 0 && <span className="text-xs bg-indigo-900 text-indigo-300 px-2 py-0.5 rounded-full">이해도 {s.score}/10</span>}
                        </div>
                        <p className="text-gray-500 text-xs">{new Date(s.createdAt).toLocaleString('ko-KR')}</p>
                        <p className="text-gray-400 text-sm mt-2 line-clamp-2">{s.explanation}</p>
                    </div>
                ))}
            </div>
        </div>
    )
}

function SessionDetail({ session, onBack }: { session: SessionRecord; onBack: () => void }) {
    return (
        <div className="max-w-2xl w-full">
            <button onClick={onBack} className="text-gray-400 hover:text-white text-sm mb-6">← 목록으로</button>
            <h2 className="text-2xl font-bold mb-1">{session.topic}</h2>
            <p className="text-gray-500 text-xs mb-6">{new Date(session.createdAt).toLocaleString('ko-KR')}</p>
            <div className="bg-gray-900 rounded-xl p-4 border border-gray-800 mb-4">
                <p className="text-xs text-gray-500 mb-2">내 설명</p>
                <p className="text-gray-300 text-sm whitespace-pre-wrap">{session.explanation}</p>
            </div>
            <div className="bg-gray-900 rounded-xl p-4 border border-gray-800">
                <p className="text-xs text-gray-500 mb-2">AI 분석</p>
                <div className="prose prose-invert prose-sm max-w-none prose-p:text-gray-300 prose-headings:text-white prose-strong:text-white prose-ul:text-gray-300 prose-li:text-gray-300 prose-code:text-green-400">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{session.analysis}</ReactMarkdown>
                </div>
            </div>
        </div>
    )
}

function App() {
    const [page, setPage] = useState<Page>('learn')
    const [step, setStep] = useState(1)
    const [topic, setTopic] = useState('')
    const [explanation, setExplanation] = useState('')
    const [analysis, setAnalysis] = useState('')
    const [sessions, setSessions] = useState<SessionRecord[]>(loadSessions)
    const [selectedSession, setSelectedSession] = useState<SessionRecord | null>(null)
    const [sessionReady, setSessionReady] = useState<boolean | null>(null)

    function saveAndFinish(score: number) {
        const record: SessionRecord = {
            id: Date.now().toString(),
            topic, explanation, analysis, score,
            createdAt: new Date().toISOString(),
        }
        saveSession(record)
        setSessions(loadSessions())
    }

    function restart() {
        setStep(1)
        setTopic('')
        setExplanation('')
        setAnalysis('')
        setSessionReady(null)
    }

    function handleTopicNext(t: string) {
        setTopic(t)
        setStep(2)
        setSessionReady(false)
        StartSession(t)
            .then(() => setSessionReady(true))
            .catch(() => setSessionReady(null))
    }

    return (
        <div className="flex h-screen bg-gray-950 text-white">
            {/* 사이드바 */}
            <aside className="w-56 bg-gray-900 border-r border-gray-800 p-4 flex flex-col">
                <h1 className="text-xl font-bold text-indigo-400 mb-4">Booktology</h1>
                <nav className="space-y-1 mb-4">
                    <div onClick={() => setPage('learn')}
                        className={`px-3 py-2 rounded-lg text-sm cursor-pointer ${page === 'learn' ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:bg-gray-800'}`}>
                        파인만 학습
                    </div>
                    <div onClick={() => { setPage('history'); setSelectedSession(null) }}
                        className={`px-3 py-2 rounded-lg text-sm cursor-pointer ${page === 'history' ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:bg-gray-800'}`}>
                        학습 기록 {sessions.length > 0 && <span className="ml-1 text-xs text-gray-500">({sessions.length})</span>}
                    </div>
                    <div onClick={() => setPage('settings')}
                        className={`px-3 py-2 rounded-lg text-sm cursor-pointer ${page === 'settings' ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:bg-gray-800'}`}>
                        설정
                    </div>
                </nav>

                {/* 최근 기록 미리보기 */}
                {sessions.length > 0 && page !== 'history' && (
                    <div className="mt-2">
                        <p className="text-xs text-gray-600 px-3 mb-2">최근 학습</p>
                        <div className="space-y-1">
                            {sessions.slice(0, 5).map(s => (
                                <div key={s.id}
                                    onClick={() => { setSelectedSession(s); setPage('history') }}
                                    className="px-3 py-1.5 rounded-lg text-xs text-gray-400 hover:bg-gray-800 cursor-pointer truncate">
                                    {s.topic}
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </aside>

            {/* 메인 */}
            <main className="flex-1 p-8 overflow-auto">
                {page === 'settings' && <Settings onBack={() => setPage('learn')} />}
                {page === 'history' && (
                    selectedSession
                        ? <SessionDetail session={selectedSession} onBack={() => setSelectedSession(null)} />
                        : <HistoryView sessions={sessions} onSelect={s => setSelectedSession(s)} />
                )}
                {page === 'learn' && (
                    <>
                        <StepIndicator current={step} />
                        {step === 1 && <Step1 onNext={handleTopicNext} />}
                        {step === 2 && <Step2 topic={topic} onNext={t => { setExplanation(t); setStep(3) }} sessionReady={sessionReady} />}
                        {step === 3 && <Step3 topic={topic} explanation={explanation}
                            onAnalyzed={a => setAnalysis(a)}
                            onNext={() => setStep(4)} />}
                        {step === 4 && <Step4 topic={topic} onRestart={restart} onSave={saveAndFinish} />}
                    </>
                )}
            </main>
        </div>
    )
}

export default App
