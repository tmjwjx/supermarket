import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { getUser, readProfile, type Profile } from '../api/user.ts'
import { clearSession, loadToken, loadUserId, saveUserId } from '../session/index.ts'

export default function ProfilePage() {
  const [id, setId] = useState(() => loadUserId())
  const [loggedIn, setLoggedIn] = useState(() => loadToken() !== '')
  const [profile, setProfile] = useState<Profile | null>(null)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  const loadMine = useCallback(async (userId: string) => {
    const token = loadToken()
    setLoggedIn(token !== '')
    setMessage('')
    setFailed(false)
    if (!token) {
      setProfile(null)
      setFailed(true)
      setMessage('请先登录')
      return
    }
    if (!userId) {
      setProfile(null)
      setFailed(true)
      setMessage('登录或注册后会带上会员编号')
      return
    }
    try {
      const body = await getUser(userId, token)
      const next = readProfile(body)
      if (!next) {
        setProfile(null)
        setFailed(true)
        setMessage('没有读到资料')
        return
      }
      setProfile(next)
      if (next.id) saveUserId(next.id)
    } catch (err) {
      setProfile(null)
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '查询失败')
    }
  }, [])

  useEffect(() => {
    const userId = loadUserId()
    if (userId && loadToken()) void loadMine(userId)
  }, [loadMine])

  function onSubmit(event: FormEvent) {
    event.preventDefault()
    void loadMine(id.trim())
  }

  function onSignOut() {
    clearSession()
    setLoggedIn(false)
    setProfile(null)
    setId('')
    setFailed(false)
    setMessage('已退出')
  }

  return (
    <form className="slip" onSubmit={onSubmit}>
      <h1>我的资料</h1>
      <p className="lead">{loggedIn ? '这是你在店里的会员资料' : '登录后查看自己的资料'}</p>
      <label>
        会员编号
        <input value={id} onChange={(e) => setId(e.target.value)} name="id" autoComplete="off" required />
      </label>
      <button className="primary" type="submit">
        查看我的资料
      </button>
      {profile ? (
        <dl className="card">
          {profile.avatarUrl ? <img className="avatar" src={profile.avatarUrl} alt="头像" /> : null}
          <dt>称呼</dt>
          <dd>{profile.nickname || '还没写称呼'}</dd>
          <dt>手机号</dt>
          <dd>{profile.phone || '未填写'}</dd>
          <dt>会员编号</dt>
          <dd>{profile.id}</dd>
          {profile.status ? (
            <>
              <dt>状态</dt>
              <dd>{profile.status}</dd>
            </>
          ) : null}
          {profile.createdAt ? (
            <>
              <dt>加入时间</dt>
              <dd>{profile.createdAt}</dd>
            </>
          ) : null}
        </dl>
      ) : null}
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
      {loggedIn ? (
        <button className="quiet" type="button" onClick={onSignOut}>
          退出
        </button>
      ) : (
        <p className="switch">
          <Link to="/login">去登录</Link>
        </p>
      )}
    </form>
  )
}
