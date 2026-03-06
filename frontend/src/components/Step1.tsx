import { useState } from 'react'

interface Props {
    onNext: (topic: string) => void
}

export function Step1({ onNext }: Props) {
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
                disabled={!topic.trim()}
                className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-700 disabled:text-gray-500 rounded-lg py-3 font-semibold transition-colors"
            >
                학습 시작 →
            </button>
        </div>
    )
}
