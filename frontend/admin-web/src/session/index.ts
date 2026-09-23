const tokenKey = 'admin_access_token'

export function saveToken(token: string) {
  sessionStorage.setItem(tokenKey, token)
}

export function loadToken() {
  return sessionStorage.getItem(tokenKey) ?? ''
}

export function clearSession() {
  sessionStorage.removeItem(tokenKey)
}

// 反斜杠会被浏览器当成斜杠 控制字符会被 URL 解析吞掉 两者都可能把路径变成外站
function hasUnsafeChar(raw: string): boolean {
  for (const ch of raw) {
    const code = ch.charCodeAt(0)
    if (ch === '\\' || code < 0x20 || code === 0x7f) return true
  }
  return false
}

// safeFrom 登录后的回跳地址 只认同源站内路径 取 pathname+search+hash 拒绝时回到商品列表
export function safeFrom(raw: string | null): string {
  const fallback = '/products'
  if (!raw || hasUnsafeChar(raw)) return fallback
  let url: URL
  try {
    url = new URL(raw, window.location.origin)
  } catch {
    return fallback
  }
  if (url.origin !== window.location.origin) return fallback
  const path = url.pathname + url.search + url.hash
  if (path.startsWith('/login')) return fallback
  return path
}
