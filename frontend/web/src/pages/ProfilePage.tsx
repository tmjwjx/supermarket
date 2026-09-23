import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listAddresses, readAddresses, setDefaultAddress, type Address } from '../api/address.ts'
import { getMe, readProfile, type Profile } from '../api/user.ts'
import { clearSession } from '../session/index.ts'

export default function ProfilePage() {
  const [profile, setProfile] = useState<Profile | null>(null)
  const [addresses, setAddresses] = useState<Address[]>([])
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let gone = false
    getMe()
      .then((body) => {
        if (gone) return
        const next = readProfile(body)
        if (!next) {
          setFailed(true)
          setMessage('没有读到资料')
          return
        }
        setProfile(next)
        listAddresses()
          .then((body) => {
            if (!gone) setAddresses(readAddresses(body))
          })
          .catch(() => {})
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '查询失败')
      })
    return () => {
      gone = true
    }
  }, [])

  function onSignOut() {
    clearSession()
    window.location.assign('/login')
  }

  return (
    <section className="slip">
      <h1>我的资料</h1>
      <p className="lead">这是你在店里的会员资料</p>
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
      {addresses.length ? (
        <ul>
          {addresses.map((item) => (
            <li key={item.id}>
              {item.receiver} {item.phone} {item.province}{item.city}{item.district}{item.detail}
              {item.isDefault ? ' 默认' : ''}
              {item.isDefault ? null : (
                <button
                  className="quiet"
                  type="button"
                  onClick={() => {
                    setDefaultAddress(item.id)
                      .then(() => listAddresses())
                      .then((body) => setAddresses(readAddresses(body)))
                      .catch((err: unknown) => {
                        setFailed(true)
                        setMessage(err instanceof Error ? err.message : '没设成默认')
                      })
                  }}
                >
                  设为默认
                </button>
              )}
            </li>
          ))}
        </ul>
      ) : null}
      {profile ? (
        <p className="switch">
          <Link to="/addresses">管理收货地址</Link>
        </p>
      ) : null}
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
      {profile ? (
        <button className="quiet" type="button" onClick={onSignOut}>
          退出
        </button>
      ) : (
        <p className="switch">
          <Link to="/login">去登录</Link>
        </p>
      )}
    </section>
  )
}
