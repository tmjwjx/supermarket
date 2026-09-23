import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { login, readAccessToken, readUserId } from '../api/user.ts'
import { saveToken, saveUserId } from '../session/index.ts'

export default function LoginPage() {
  const navigate = useNavigate()
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setMessage('')
    setFailed(false)
    try {
      const body = await login({ phone, password })
      const token = readAccessToken(body)
      if (!token) {
        setFailed(true)
        setMessage('登录成功，响应里没有 access_token')
        return
      }
      saveToken(token)
      saveUserId(readUserId(body))
      navigate('/me')
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '登录失败')
    }
  }

  return (
    <form className="slip" onSubmit={onSubmit}>
      <h1>登录</h1>
      <p className="lead">用手机号进入街口超市</p>
      <label>
        手机号
        <input value={phone} onChange={(e) => setPhone(e.target.value)} name="phone" inputMode="tel" autoComplete="tel" required />
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
      <button className="primary" type="submit">
        登录
      </button>
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
      <p className="switch">
        <Link to="/register">办理会员</Link>
      </p>
    </form>
  )
}
