import { useState } from 'react'
import { StartSession } from '../wailsjs/go/main/App'
import { StepIndicator } from './components/StepIndicator'
import { Step1 } from './components/Step1'
import { Step2 } from './components/Step2'
import { Step3 } from './components/Step3'
import { Step4 } from './components/Step4'
import { Settings } from './components/Settings'
import { HistoryView } from './components/HistoryView'
import { SessionDetail } from './components/SessionDetail'
import { useSessions } from './hooks/useSessions'
import type { Page, SessionRecord } from './types'

export default function App() {
    const [page, setPage] = useState<Page>('learn')
    const [step, setStep] = useState(1)
    const [topic, setTopic] = useState('')
    const [explanation, setExplanation] = useState('')
    const [analysis, setAnalysis] = useState('')
    const [sessionReady, setSessionReady] = useState<boolean | null>(null)
    const [selectedSession, setSelectedSession] = useState<SessionRecord | null>(null)
    const { sessions, save: saveSession } = useSessions()

    function handleTopicNext(t: string) {
        setTopic(t)
        setStep(2)
        setSessionReady(false)
        StartSession(t)
            .then(() => setSessionReady(true))
            .catch(() => setSessionReady(null))
    }

    function handleSaveAndFinish(score: number) {
        saveSession({
            id: Date.now().toString(),
            topic, explanation, analysis, score,
            createdAt: new Date().toISOString(),
        })
    }

    function restart() {
        setStep(1)
        setTopic('')
        setExplanation('')
        setAnalysis('')
        setSessionReady(null)
    }

    return (
        <div className="flex h-screen bg-gray-950 text-white">
            {/* 사이드바 */}
            <aside className="w-56 bg-gray-900 border-r border-gray-800 p-4 flex flex-col">
                <h1 className="text-xl font-bold text-indigo-400 mb-4">Booktology</h1>
                <nav className="space-y-1 mb-4">
                    {([
                        { id: 'learn', label: '파인만 학습' },
                        { id: 'history', label: `학습 기록${sessions.length > 0 ? ` (${sessions.length})` : ''}` },
                        { id: 'settings', label: '설정' },
                    ] as const).map(item => (
                        <div
                            key={item.id}
                            onClick={() => { setPage(item.id); if (item.id === 'history') setSelectedSession(null) }}
                            className={`px-3 py-2 rounded-lg text-sm cursor-pointer ${page === item.id ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:bg-gray-800'}`}
                        >
                            {item.label}
                        </div>
                    ))}
                </nav>

                {sessions.length > 0 && page !== 'history' && (
                    <div className="mt-2">
                        <p className="text-xs text-gray-600 px-3 mb-2">최근 학습</p>
                        <div className="space-y-1">
                            {sessions.slice(0, 5).map(s => (
                                <div
                                    key={s.id}
                                    onClick={() => { setSelectedSession(s); setPage('history') }}
                                    className="px-3 py-1.5 rounded-lg text-xs text-gray-400 hover:bg-gray-800 cursor-pointer truncate"
                                >
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
                        : <HistoryView sessions={sessions} onSelect={setSelectedSession} />
                )}

                {page === 'learn' && (
                    <>
                        <StepIndicator current={step} />
                        {step === 1 && <Step1 onNext={handleTopicNext} />}
                        {step === 2 && (
                            <Step2
                                topic={topic}
                                sessionReady={sessionReady}
                                onNext={text => { setExplanation(text); setStep(3) }}
                            />
                        )}
                        {step === 3 && (
                            <Step3
                                topic={topic}
                                explanation={explanation}
                                onAnalyzed={setAnalysis}
                                onNext={() => setStep(4)}
                            />
                        )}
                        {step === 4 && (
                            <Step4
                                topic={topic}
                                onRestart={restart}
                                onSave={handleSaveAndFinish}
                            />
                        )}
                    </>
                )}
            </main>
        </div>
    )
}
