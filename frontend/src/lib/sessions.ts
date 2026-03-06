import type { SessionRecord } from '../types'

const STORAGE_KEY = 'booktology-sessions'
const MAX_SESSIONS = 50

export function loadSessions(): SessionRecord[] {
    try {
        return JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]')
    } catch {
        return []
    }
}

export function saveSession(record: SessionRecord): void {
    const sessions = loadSessions()
    localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify([record, ...sessions].slice(0, MAX_SESSIONS))
    )
}
