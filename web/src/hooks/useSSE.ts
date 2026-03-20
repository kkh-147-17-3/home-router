import { useState, useEffect, useRef, useCallback } from 'react'

interface UseSSEOptions {
  enabled?: boolean
  maxItems?: number
}

export function useSSE<T>(url: string, options?: UseSSEOptions) {
  const { enabled = true, maxItems = 500 } = options || {}
  const [data, setData] = useState<T[]>([])
  const [connected, setConnected] = useState(false)
  const esRef = useRef<EventSource | null>(null)

  useEffect(() => {
    if (!enabled) {
      if (esRef.current) {
        esRef.current.close()
        esRef.current = null
        setConnected(false)
      }
      return
    }

    const es = new EventSource(url)
    esRef.current = es

    es.onopen = () => setConnected(true)
    es.onmessage = (e) => {
      try {
        const parsed = JSON.parse(e.data) as T
        setData(prev => [...prev.slice(-(maxItems - 1)), parsed])
      } catch { /* ignore parse errors */ }
    }
    es.onerror = () => setConnected(false)

    return () => {
      es.close()
      esRef.current = null
      setConnected(false)
    }
  }, [url, enabled, maxItems])

  const clear = useCallback(() => setData([]), [])

  return { data, connected, clear }
}
