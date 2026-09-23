import { clearSession, loadToken } from '../session/index.ts'

const apiBase = import.meta.env.VITE_API_BASE ?? ''

export async function send(path: string, init: RequestInit = {}): Promise<unknown> {
  const headers = new Headers(init.headers)
  const token = loadToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  let res: Response
  try {
    res = await fetch(`${apiBase}${path}`, { ...init, headers })
  } catch {
    throw new Error('连不上网关')
  }
  const text = await res.text()
  let body: unknown = null
  if (text) {
    try {
      body = JSON.parse(text) as unknown
    } catch {
      body = text
    }
  }
  if (res.status === 401) {
    clearSession()
    if (window.location.pathname !== '/login') {
      const next = `${window.location.pathname}${window.location.search}`
      window.location.assign(`/login?from=${encodeURIComponent(next)}`)
    }
  }
  if (!res.ok) throw new Error(messageOf(body, res.status))
  return body
}

function messageOf(body: unknown, status: number): string {
  if (body && typeof body === 'object' && 'message' in body) {
    const message = body.message
    if (typeof message === 'string' && message !== '') return message
  }
  if (typeof body === 'string' && body !== '') return body
  return `请求失败 ${status}`
}
