import { useState, type FormEvent } from 'react'
import { getUser } from '../api.ts'
import { loadToken } from '../session.ts'

export default function UserPage() {
  const [id, setId] = useState('')
  const [message, setMessage] = useState('')
  const [result, setResult] = useState('')
  const token = loadToken()

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setMessage('')
    setResult('')
    try {
      const body = await getUser(id, loadToken())
      setMessage('查询成功')
      setResult(JSON.stringify(body, null, 2))
    } catch (err) {
      setMessage(err instanceof Error ? err.message : '查询失败')
    }
  }

  return (
    <form onSubmit={onSubmit}>
      <h1>按 id 查资料</h1>
      <p className="message">{token ? '已持有 access_token' : '尚未登录，请求不会带 Authorization'}</p>
      <label>
        用户 id
        <input value={id} onChange={(e) => setId(e.target.value)} name="id" required />
      </label>
      <button type="submit">查询</button>
      {message ? <p className="message">{message}</p> : null}
      {result ? <pre>{result}</pre> : null}
    </form>
  )
}
