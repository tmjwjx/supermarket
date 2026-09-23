// 只请求 gateway 开发时 Vite 把 /v1 代理到 127.0.0.1:8080
const apiBase = import.meta.env.VITE_API_BASE ?? ''

export async function send(path: string, init: RequestInit): Promise<unknown> {
  let res: Response
  try {
    res = await fetch(`${apiBase}${path}`, init)
  } catch {
    throw new Error('现在连不上店')
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
