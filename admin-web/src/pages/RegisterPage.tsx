import { useState, type FormEvent } from 'react'
import { readAccessToken, register } from '../api.ts'
import { saveToken } from '../session.ts'

export default function RegisterPage() {
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [message, setMessage] = useState('')
  const [result, setResult] = useState('')

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setMessage('')
    setResult('')
    try {
      const body = await register({ phone, password, nickname })
      const token = readAccessToken(body)
      if (token) saveToken(token)
      setMessage(token ? '注册成功，已保存 access_token' : '注册成功')
      setResult(JSON.stringify(body, null, 2))
    } catch (err) {
      setMessage(err instanceof Error ? err.message : '注册失败')
    }
  }

  return (
    <form onSubmit={onSubmit}>
      <h1>注册</h1>
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
          autoComplete="new-password"
          required
        />
      </label>
      <label>
        昵称
        <input value={nickname} onChange={(e) => setNickname(e.target.value)} name="nickname" autoComplete="nickname" />
      </label>
      <button type="submit">注册</button>
      {message ? <p className="message">{message}</p> : null}
      {result ? <pre>{result}</pre> : null}
    </form>
  )
}
