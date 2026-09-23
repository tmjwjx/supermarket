// access_token 与用户编号放 sessionStorage 不解析 JWT 也不刷新
const tokenKey = 'access_token'
const userIdKey = 'user_id'

export function saveToken(token: string) {
  sessionStorage.setItem(tokenKey, token)
}

export function loadToken() {
  return sessionStorage.getItem(tokenKey) ?? ''
}

export function saveUserId(id: string) {
  if (!id) return
  sessionStorage.setItem(userIdKey, id)
}

export function loadUserId() {
  return sessionStorage.getItem(userIdKey) ?? ''
}

export function clearSession() {
  sessionStorage.removeItem(tokenKey)
  sessionStorage.removeItem(userIdKey)
}

// 反斜杠会被浏览器当成斜杠 控制字符会被 URL 解析吞掉 两者都可能把路径变成外站
function hasUnsafeChar(raw: string): boolean {
  for (const ch of raw) {
    const code = ch.charCodeAt(0)
    if (ch === '\\' || code < 0x20 || code === 0x7f) return true
  }
  return false
}

// sameOriginPath 只认解析后与本站同源的地址 返回 pathname+search+hash 其余返回空串
export function sameOriginPath(raw: string | null): string {
  if (!raw || hasUnsafeChar(raw)) return ''
  let url: URL
  try {
    url = new URL(raw, window.location.origin)
  } catch {
    return ''
  }
  if (url.origin !== window.location.origin) return ''
  return url.pathname + url.search + url.hash
}

// safeFrom 登录注册后的回跳地址 不同源或指回登录注册页时回到我的
export function safeFrom(raw: string | null): string {
  const path = sameOriginPath(raw)
  if (!path || path.startsWith('/login') || path.startsWith('/register')) {
    return '/me'
  }
  return path
}
