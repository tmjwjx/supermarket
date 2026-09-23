const apiBase = import.meta.env.VITE_API_BASE ?? ''

export function readAccessToken(body: unknown): string {
  if (!body || typeof body !== 'object') return ''
  const record = body as { access_token?: unknown; accessToken?: unknown }
  const token = record.access_token ?? record.accessToken
  return typeof token === 'string' ? token : ''
}

export function register(input: { phone: string; password: string; nickname: string }) {
  const payload: Record<string, string> = {
    phone: input.phone,
    password: input.password,
  }
  if (input.nickname) payload.nickname = input.nickname
  return send('/v1/users/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

export function login(input: { phone: string; password: string }) {
  return send('/v1/users/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      phone: input.phone,
      password: input.password,
    }),
  })
}

export function getUser(id: string, token: string) {
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  return send(`/v1/users/${encodeURIComponent(id)}`, { headers })
}

async function send(path: string, init: RequestInit): Promise<unknown> {
  const res = await fetch(`${apiBase}${path}`, init)
  const text = await res.text()
  let body: unknown = null
  if (text) {
    try {
      body = JSON.parse(text) as unknown
    } catch {
      body = text
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
