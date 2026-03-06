import { useState } from 'react'

interface Props {
    topic: string
    sessionReady: boolean | null
    onNext: (text: string) => void
}

export function Step2({ topic, sessionReady, onNext }: Props) {
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
