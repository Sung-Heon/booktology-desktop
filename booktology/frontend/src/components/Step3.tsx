import { useState, useEffect, useRef } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { AnalyzeStreaming, ChatStreaming } from '../../wailsjs/go/main/App'
import { useStreaming } from '../hooks/useStreaming'
import { PROSE } from '../lib/constants'
import type { ChatMsg } from '../types'

interface Props {
    topic: string
    explanation: string
    onAnalyzed: (analysis: string) => void
    onNext: () => void
}

export function Step3({ topic, explanation, onAnalyzed, onNext }: Props) {
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')
    const [messages, setMessages] = useState<ChatMsg[]>([
        { role: 'user', content: explanation },
        { role: 'assistant', content: '', streaming: true },
    ])
    const [chatInput, setChatInput] = useState('')
    const [chatLoading, setChatLoading] = useState(false)
    const chatEndRef = useRef<HTMLDivElement>(null)
    const accRef = useRef('')
    const { listen, stop } = useStreaming()

    // 초기 분석 스트리밍
    useEffect(() => {
        accRef.current = ''
        listen({
            onChunk: chunk => {
                accRef.current += chunk
                setMessages(prev => {
                    const last = prev[prev.length - 1]
                    if (last?.streaming) return [...prev.slice(0, -1), { ...last, content: accRef.current }]
                    return prev
                })
            },
            onDone: () => {
                setMessages(prev => prev.map((m, i) =>
                    i === prev.length - 1 ? { ...m, content: accRef.current, streaming: false } : m
                ))
                onAnalyzed(accRef.current)
                setLoading(false)
            },
            onError: err => {
                setError(err)
                setLoading(false)
            },
        })
        AnalyzeStreaming(topic, explanation)
        return stop
    }, [])

    useEffect(() => {
        chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, [messages, chatLoading])

    function sendMessage() {
        const msg = chatInput.trim()
        if (!msg || chatLoading) return
        const history = [...messages, { role: 'user' as const, content: msg }]
        setMessages([...history, { role: 'assistant' as const, content: '', streaming: true }])
        setChatInput('')
        setChatLoading(true)

        listen({
            onChunk: chunk => {
                setMessages(prev => {
                    const last = prev[prev.length - 1]
                    if (last?.streaming) return [...prev.slice(0, -1), { ...last, content: last.content + chunk }]
                    return prev
                })
            },
            onDone: () => {
                setMessages(prev => prev.map((m, i) => i === prev.length - 1 ? { ...m, streaming: false } : m))
                setChatLoading(false)
            },
            onError: err => {
                setMessages(prev => [...prev.slice(0, -1), { role: 'assistant', content: `오류: ${err}`, streaming: false }])
                setChatLoading(false)
            },
        })
        ChatStreaming(history, msg)
    }

    return (
        <div className="max-w-2xl w-full flex flex-col" style={{ height: 'calc(100vh - 160px)' }}>
            <h2 className="text-2xl font-bold mb-1">AI 분석 & 대화</h2>
            <p className="text-gray-400 mb-3 text-sm">분석 결과를 보고 추가 질문을 이어갈 수 있어요.</p>

            <div className="flex-1 overflow-auto bg-gray-900 rounded-xl border border-gray-800 p-4 space-y-4 mb-3">
                {error && <p className="text-red-400 text-sm">오류: {error}</p>}
                {messages.map((msg, i) => (
                    <div key={i} className={`flex gap-2 ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                        {msg.role === 'assistant' && (
                            <div className="w-7 h-7 rounded-full bg-indigo-700 flex items-center justify-center text-xs font-bold shrink-0 mt-1">AI</div>
                        )}
                        <div className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm
                            ${msg.role === 'user'
                                ? 'bg-indigo-600 text-white rounded-tr-sm'
                                : 'bg-gray-800 text-gray-200 rounded-tl-sm border border-gray-700'}`}>
                            {msg.role === 'assistant' ? (
                                msg.streaming && !msg.content ? (
                                    <div className="flex items-center gap-2 text-gray-400 py-1">
                                        <div className="w-3 h-3 rounded-full border-2 border-indigo-400 border-t-transparent animate-spin shrink-0" />
                                        <span className="text-xs">응답 생성 중...</span>
                                    </div>
                                ) : msg.content ? (
                                    <div className={PROSE}>
                                        <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
                                    </div>
                                ) : null
                            ) : msg.content}
                        </div>
                        {msg.role === 'user' && (
                            <div className="w-7 h-7 rounded-full bg-gray-600 flex items-center justify-center text-xs shrink-0 mt-1">나</div>
                        )}
                    </div>
                ))}
                <div ref={chatEndRef} />
            </div>

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
