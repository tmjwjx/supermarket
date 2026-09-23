import { useCallback, useEffect, useRef, useState } from 'react'

export function errText(err: unknown, fallback: string): string {
  return err instanceof Error && err.message ? err.message : fallback
}

export function formatTime(unix: number): string {
  if (!unix) return '-'
  const d = new Date(unix * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// datetime-local 控件与 unix 秒互转 空值记为 0
export function unixToLocalInput(unix: number): string {
  if (!unix) return ''
  const d = new Date(unix * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function localInputToUnix(text: string): number {
  if (!text) return 0
  const ms = new Date(text).getTime()
  return Number.isFinite(ms) ? Math.floor(ms / 1000) : 0
}

// deps 变化或调用 reload 时重新加载 过期的结果直接丢弃
export function useLoad<T>(load: () => Promise<T>, initial: T, deps: unknown[] = []) {
  const [data, setData] = useState<T>(initial)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [tick, setTick] = useState(0)
  const loadRef = useRef(load)
  const key = JSON.stringify(deps)
  useEffect(() => {
    loadRef.current = load
  })
  useEffect(() => {
    let gone = false
    setLoading(true)
    loadRef.current()
      .then((value) => {
        if (gone) return
        setData(value)
        setError('')
      })
      .catch((err: unknown) => {
        if (!gone) setError(errText(err, '加载失败'))
      })
      .finally(() => {
        if (!gone) setLoading(false)
      })
    return () => {
      gone = true
    }
  }, [key, tick])
  const reload = useCallback(() => setTick((n) => n + 1), [])
  return { data, setData, error, loading, reload }
}

type Note = { text: string; tone: 'info' | 'error' }

export function useNote() {
  const [note, setNote] = useState<Note>({ text: '', tone: 'info' })
  const ok = useCallback((text: string) => setNote({ text, tone: 'info' }), [])
  const fail = useCallback((err: unknown, fallback: string) => setNote({ text: errText(err, fallback), tone: 'error' }), [])
  const clear = useCallback(() => setNote({ text: '', tone: 'info' }), [])
  return { note, ok, fail, clear }
}
