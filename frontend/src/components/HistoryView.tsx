import type { SessionRecord } from '../types'

interface Props {
    sessions: SessionRecord[]
    onSelect: (s: SessionRecord) => void
}

export function HistoryView({ sessions, onSelect }: Props) {
    if (sessions.length === 0) {
        return <div className="text-gray-500 text-sm mt-8">아직 학습 기록이 없어요.</div>
    }
    return (
        <div className="max-w-2xl w-full">
            <h2 className="text-2xl font-bold mb-6">학습 기록</h2>
            <div className="space-y-3">
                {sessions.map(s => (
                    <div
                        key={s.id}
                        onClick={() => onSelect(s)}
                        className="bg-gray-900 rounded-xl p-4 border border-gray-800 hover:border-indigo-500 cursor-pointer transition-colors"
                    >
                        <div className="flex items-center justify-between mb-1">
                            <span className="font-semibold text-white">{s.topic}</span>
                            {s.score > 0 && (
                                <span className="text-xs bg-indigo-900 text-indigo-300 px-2 py-0.5 rounded-full">
                                    이해도 {s.score}/10
                                </span>
                            )}
                        </div>
                        <p className="text-gray-500 text-xs">{new Date(s.createdAt).toLocaleString('ko-KR')}</p>
                        <p className="text-gray-400 text-sm mt-2 line-clamp-2">{s.explanation}</p>
                    </div>
                ))}
            </div>
        </div>
    )
}
