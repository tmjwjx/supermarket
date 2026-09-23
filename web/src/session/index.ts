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
