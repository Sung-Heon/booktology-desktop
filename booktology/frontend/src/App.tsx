function App() {
    return (
        <div className="flex h-screen bg-gray-950 text-white">
            {/* 사이드바 */}
            <aside className="w-64 bg-gray-900 border-r border-gray-800 p-4">
                <h1 className="text-xl font-bold text-indigo-400 mb-6">Booktology</h1>
                <nav className="space-y-2">
                    <div className="px-3 py-2 rounded-lg bg-indigo-600 text-white text-sm">대시보드</div>
                    <div className="px-3 py-2 rounded-lg text-gray-400 hover:bg-gray-800 text-sm cursor-pointer">새 학습 세션</div>
                </nav>
            </aside>

            {/* 메인 영역 */}
            <main className="flex-1 p-8">
                <h2 className="text-2xl font-bold mb-2">파인만 학습법</h2>
                <p className="text-gray-400 mb-8">개념을 선택하고 학습을 시작하세요.</p>

                <div className="bg-gray-900 rounded-xl p-6 border border-gray-800 max-w-xl">
                    <label className="block text-sm text-gray-400 mb-2">학습할 개념</label>
                    <input
                        type="text"
                        placeholder="예) 양자역학, 이진탐색, 광합성..."
                        className="w-full bg-gray-800 rounded-lg px-4 py-3 text-white placeholder-gray-500 outline-none focus:ring-2 focus:ring-indigo-500"
                    />
                    <button className="mt-4 w-full bg-indigo-600 hover:bg-indigo-700 rounded-lg py-3 font-semibold transition-colors">
                        학습 시작 →
                    </button>
                </div>
            </main>
        </div>
    )
}

export default App
