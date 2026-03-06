import { useState } from 'react'

interface Props {
    topic: string
    onRestart: () => void
    onSave: (score: number) => void
}

export function Step4({ topic, onRestart, onSave }: Props) {
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
