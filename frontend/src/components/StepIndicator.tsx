import { STEPS } from '../lib/constants'

export function StepIndicator({ current }: { current: number }) {
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
