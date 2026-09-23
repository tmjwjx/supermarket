import { useState, type FormEvent } from 'react'
import { login, readAccessToken } from '../api.ts'
import { saveToken } from '../session.ts'

export default function LoginPage() {
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [message, setMessage] = useState('')
  const [result, setResult] = useState('')

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setMessage('')
    setResult('')
    try {
      const body = await login({ phone, password })
      const token = readAccessToken(body)
      if (token) saveToken(token)
      setMessage(token ? '登录成功，已保存 access_token' : '登录成功，响应里没有 access_token')
      setResult(JSON.stringify(body, null, 2))
    } catch (err) {
      setMessage(err instanceof Error ? err.message : '登录失败')
    }
  }

  return (
    <form onSubmit={onSubmit}>
      <h1>登录</h1>
      <label>
        手机号
        <input value={phone} onChange={(e) => setPhone(e.target.value)} name="phone" autoComplete="tel" required />
      </label>
      <label>
        密码
        <input
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          name="password"
          type="password"
          autoComplete="current-password"
          required
        />
      </label>
      <button type="submit">登录</button>
      {message ? <p className="message">{message}</p> : null}
      {result ? <pre>{result}</pre> : null}
    </form>
  )
}
