import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { SessionRecord } from '../types'

interface Props {
    session: SessionRecord
    onBack: () => void
}

export function SessionDetail({ session, onBack }: Props) {
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
