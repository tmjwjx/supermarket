import { useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { listNotifications, readNotifications } from './api/notice.ts'
import AppRoutes from './routes.tsx'
import { loadToken } from './session/index.ts'

export default function App() {
  return (
    <div className="store">
      <header className="sign">
        <p className="brand">街口超市</p>
        <nav>
          <NavLink to="/" end>
            首页
          </NavLink>
          <NavLink to="/cart">购物车</NavLink>
          <NavLink to="/orders">订单</NavLink>
          <NavLink to="/favorites">收藏</NavLink>
          <NavLink to="/history">足迹</NavLink>
          <MessageLink />
          <NavLink to="/me">我的</NavLink>
          <NavLink to="/login">登录</NavLink>
          <NavLink to="/register">注册</NavLink>
        </nav>
      </header>
      <div className="stripe" />
      <main>
        <AppRoutes />
      </main>
    </div>
  )
}

function MessageLink() {
  const [unread, setUnread] = useState(0)
  useEffect(() => {
    if (!loadToken()) return
    let gone = false
    listNotifications()
      .then((body) => {
        if (!gone) setUnread(readNotifications(body).unread)
      })
      .catch(() => {})
    return () => {
      gone = true
    }
  }, [])
  return <NavLink to="/messages">{unread > 0 ? `消息 ${unread}` : '消息'}</NavLink>
}
