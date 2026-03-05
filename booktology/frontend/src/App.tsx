import { useState, useEffect } from 'react'
import { AnalyzeExplanation, SetProvider, ConnectChatGPTOAuth, GetProviderType } from '../wailsjs/go/main/App'

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

function Step2({ topic, onNext }: { topic: string; onNext: (text: string) => void }) {
    const [text, setText] = useState('')
    return (
        <div className="max-w-2xl w-full">
            <h2 className="text-2xl font-bold mb-2">"{topic}"을 설명해보세요</h2>
            <p className="text-gray-400 mb-6">초등학생에게 설명한다고 생각하고 자유롭게 써보세요. 틀려도 괜찮아요.</p>
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

function Step3({ topic, explanation, onNext }: { topic: string; explanation: string; onNext: () => void }) {
    const [result, setResult] = useState('')
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')

    useEffect(() => {
        AnalyzeExplanation(topic, explanation)
            .then(res => setResult(res))
            .catch(err => setError(String(err)))
            .finally(() => setLoading(false))
    }, [])

    return (
        <div className="max-w-2xl w-full">
            <h2 className="text-2xl font-bold mb-2">Claude 분석 결과</h2>
            <p className="text-gray-400 mb-6">이해 갭과 보완이 필요한 부분이에요.</p>
            <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 mb-4 text-gray-300 text-sm leading-relaxed min-h-40">
                {loading && <p className="text-gray-500 animate-pulse">Claude가 분석 중...</p>}
                {error && <p className="text-red-400">오류: {error}</p>}
                {result && <p className="whitespace-pre-wrap">{result}</p>}
            </div>
            <button
                onClick={onNext}
                disabled={loading}
                className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-700 disabled:text-gray-500 rounded-lg py-3 font-semibold transition-colors"
            >
                복습하기 →
            </button>
        </div>
    )
}

function Step4({ topic, onRestart }: { topic: string; onRestart: () => void }) {
    const [score, setScore] = useState(0)
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
            <button
                onClick={onRestart}
                className="w-full bg-gray-700 hover:bg-gray-600 rounded-lg py-3 font-semibold transition-colors"
            >
                새 개념 학습하기
            </button>
        </div>
    )
}

type ProviderType = 'claude-cli' | 'anthropic' | 'openai' | 'chatgpt-oauth'

function Settings({ onBack }: { onBack: () => void }) {
    const [provider, setProvider] = useState<ProviderType>('claude-cli')
    const [activeProvider, setActiveProvider] = useState<ProviderType>('claude-cli')
    const [apiKey, setApiKey] = useState('')
    const [saved, setSaved] = useState(false)
    const [error, setError] = useState('')
    const [connecting, setConnecting] = useState(false)

    useEffect(() => {
        GetProviderType().then(p => {
            setActiveProvider(p as ProviderType)
            setProvider(p as ProviderType)
        })
    }, [])

    function save() {
        SetProvider(provider, apiKey)
            .then(() => { setSaved(true); setError(''); setActiveProvider(provider) })
            .catch(err => setError(String(err)))
    }

    function connectOAuth() {
        setConnecting(true)
        setError('')
        ConnectChatGPTOAuth()
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
                    { id: 'claude-cli', label: 'Claude Code CLI', desc: '로컬 claude CLI 사용 (느림, API 키 불필요)' },
                    { id: 'anthropic', label: 'Anthropic API', desc: 'Claude API 직접 호출 (빠름, API 키 필요)' },
                    { id: 'openai', label: 'OpenAI API 키', desc: 'ChatGPT API 키로 연결' },
                    { id: 'chatgpt-oauth', label: 'ChatGPT OAuth', desc: 'ChatGPT Plus/Pro 구독으로 브라우저 로그인' },
                ] as const).map(p => (
                    <div
                        key={p.id}
                        onClick={() => { setProvider(p.id); setApiKey(''); setSaved(false) }}
                        className={`p-4 rounded-xl border cursor-pointer transition-colors
                            ${provider === p.id ? 'border-indigo-500 bg-indigo-950' : 'border-gray-700 bg-gray-800 hover:border-gray-600'}`}
                    >
                        <div className="flex items-center gap-2">
                            <span className="font-medium">{p.label}</span>
                            {activeProvider === p.id && (
                                <span className="text-xs bg-green-600 text-white px-2 py-0.5 rounded-full">활성</span>
                            )}
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

            {error && <p className="text-red-400 text-sm mb-3">{error}</p>}
            {saved && <p className="text-green-400 text-sm mb-3">연결됐어요!</p>}

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

type Page = 'learn' | 'settings'

function App() {
    const [page, setPage] = useState<Page>('learn')
    const [step, setStep] = useState(1)
    const [topic, setTopic] = useState('')
    const [explanation, setExplanation] = useState('')

    function restart() {
        setStep(1)
        setTopic('')
        setExplanation('')
    }

    return (
        <div className="flex h-screen bg-gray-950 text-white">
            {/* 사이드바 */}
            <aside className="w-56 bg-gray-900 border-r border-gray-800 p-4 flex flex-col">
                <h1 className="text-xl font-bold text-indigo-400 mb-6">Booktology</h1>
                <nav className="space-y-1">
                    <div
                        onClick={() => setPage('learn')}
                        className={`px-3 py-2 rounded-lg text-sm cursor-pointer ${page === 'learn' ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:bg-gray-800'}`}
                    >파인만 학습</div>
                    <div
                        onClick={() => setPage('settings')}
                        className={`px-3 py-2 rounded-lg text-sm cursor-pointer ${page === 'settings' ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:bg-gray-800'}`}
                    >설정</div>
                </nav>
            </aside>

            {/* 메인 */}
            <main className="flex-1 p-8 overflow-auto">
                {page === 'settings' && <Settings onBack={() => setPage('learn')} />}
                {page === 'learn' && (
                    <>
                        <StepIndicator current={step} />
                        {step === 1 && <Step1 onNext={t => { setTopic(t); setStep(2) }} />}
                        {step === 2 && <Step2 topic={topic} onNext={t => { setExplanation(t); setStep(3) }} />}
                        {step === 3 && <Step3 topic={topic} explanation={explanation} onNext={() => setStep(4)} />}
                        {step === 4 && <Step4 topic={topic} onRestart={restart} />}
                    </>
                )}
            </main>
        </div>
    )
}

export default App
