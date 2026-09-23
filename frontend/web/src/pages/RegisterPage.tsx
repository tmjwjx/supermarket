import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { readAccessToken, readUserId, register } from '../api/user.ts'
import { saveToken, saveUserId } from '../session/index.ts'

export default function RegisterPage() {
  const navigate = useNavigate()
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setMessage('')
    setFailed(false)
    try {
      const body = await register({ phone, password, nickname })
      const token = readAccessToken(body)
      saveUserId(readUserId(body))
      if (token) {
        saveToken(token)
        navigate('/me')
        return
      }
      setMessage('注册成功')
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '注册失败')
    }
  }

  return (
    <form className="slip" onSubmit={onSubmit}>
      <h1>办理会员</h1>
      <p className="lead">留下手机号就能进店</p>
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
          autoComplete="new-password"
          required
        />
      </label>
      <label>
        怎么称呼
        <input value={nickname} onChange={(e) => setNickname(e.target.value)} name="nickname" autoComplete="nickname" />
      </label>
      <button className="primary" type="submit">
        注册
      </button>
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
      <p className="switch">
        <Link to="/login">已有会员 去登录</Link>
      </p>
    </form>
  )
}
