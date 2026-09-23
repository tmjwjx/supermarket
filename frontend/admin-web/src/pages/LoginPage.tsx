import { useState, type FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { login, readAccessToken } from '../api/admin.ts'
import { safeFrom, saveToken } from '../session/index.ts'

export default function LoginPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [message, setMessage] = useState('')

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setMessage('')
    try {
      const body = await login(username, password)
      const token = readAccessToken(body)
      if (!token) {
        setMessage('登录成功，响应里没有令牌')
        return
      }
      saveToken(token)
      navigate(safeFrom(params.get('from')))
    } catch (err) {
      setMessage(err instanceof Error ? err.message : '登录失败')
    }
  }

  return (
    <form onSubmit={onSubmit}>
      <h1>运营登录</h1>
      <label>
        用户名
        <input value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" />
      </label>
      <label>
        密码
        <input
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete="current-password"
        />
      </label>
      <button type="submit">登录</button>
      {message ? <p className="message">{message}</p> : null}
    </form>
  )
}
