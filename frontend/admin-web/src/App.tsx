import { useEffect, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { adminRoleText, getMe } from './api/admin.ts'
import AppRoutes from './routes.tsx'
import { clearSession, loadToken } from './session/index.ts'

export default function App() {
  // 订阅路由变化 登录跳转后重新读取令牌
  useLocation()
  const signedIn = loadToken() !== ''
  const [me, setMe] = useState('')

  useEffect(() => {
    if (!signedIn) return
    let gone = false
    getMe()
      .then((user) => {
        if (!gone) setMe(`${user.displayName || user.username} ${adminRoleText[user.role] ?? user.role}`)
      })
      .catch(() => {})
    return () => {
      gone = true
    }
  }, [signedIn])

  return (
    <>
      <nav>
        <NavLink to="/products" end>商品</NavLink>
        <NavLink to="/audits">审核</NavLink>
        <NavLink to="/recycle">回收站</NavLink>
        <NavLink to="/attributes">属性模板</NavLink>
        <NavLink to="/categories">分类</NavLink>
        <NavLink to="/brands">品牌</NavLink>
        <NavLink to="/slots">推荐位</NavLink>
        <NavLink to="/stocks">库存</NavLink>
        <NavLink to="/logs">日志</NavLink>
        <NavLink to="/orders">订单</NavLink>
        <NavLink to="/ship">发货</NavLink>
        <NavLink to="/reconcile">对账</NavLink>
        <NavLink to="/admin-users">运营账号</NavLink>
        {signedIn ? (
          <>
            {me ? <span className="hint">{me}</span> : null}
            <button
              type="button"
              onClick={() => {
                clearSession()
                window.location.assign('/login')
              }}
            >
              退出
            </button>
          </>
        ) : (
          <NavLink to="/login">登录</NavLink>
        )}
      </nav>
      <main>
        <AppRoutes />
      </main>
    </>
  )
}
