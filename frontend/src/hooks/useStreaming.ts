import { useRef, useCallback } from 'react'
import { EventsOn } from '../../wailsjs/runtime/runtime'

type StreamHandlers = {
    onChunk: (chunk: string) => void
    onDone: () => void
    onError: (err: string) => void
}

export function useStreaming() {
    const cleanupRef = useRef<(() => void) | null>(null)

    const listen = useCallback((handlers: StreamHandlers) => {
        // 이전 리스너 정리
        cleanupRef.current?.()

        const offChunk = EventsOn('stream:chunk', handlers.onChunk)
        const off = () => { offChunk(); offDone(); offError() }

        const offDone = EventsOn('stream:done', () => {
            handlers.onDone()
            off()
            cleanupRef.current = null
        })
        const offError = EventsOn('stream:error', (err: string) => {
            handlers.onError(err)
            off()
            cleanupRef.current = null
        })

        cleanupRef.current = off
        return off
    }, [])

    const stop = useCallback(() => {
        cleanupRef.current?.()
        cleanupRef.current = null
    }, [])

    return { listen, stop }
}
