const tokenKey = 'access_token'

export function saveToken(token: string) {
  sessionStorage.setItem(tokenKey, token)
}

export function loadToken() {
  return sessionStorage.getItem(tokenKey) ?? ''
}
