import { NavLink } from 'react-router-dom'
import AppRoutes from './routes.tsx'

export default function App() {
  return (
    <div className="store">
      <header className="sign">
        <p className="brand">街口超市</p>
        <nav>
          <NavLink to="/login">登录</NavLink>
          <NavLink to="/register">注册</NavLink>
          <NavLink to="/me">我的</NavLink>
        </nav>
      </header>
      <div className="stripe" />
      <main>
        <AppRoutes />
      </main>
    </div>
  )
}
