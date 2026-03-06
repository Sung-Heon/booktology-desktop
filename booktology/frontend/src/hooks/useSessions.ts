import { useState, useCallback } from 'react'
import { loadSessions, saveSession } from '../lib/sessions'
import type { SessionRecord } from '../types'

export function useSessions() {
    const [sessions, setSessions] = useState<SessionRecord[]>(loadSessions)

    const save = useCallback((record: SessionRecord) => {
        saveSession(record)
        setSessions(loadSessions())
    }, [])

    return { sessions, save }
}
