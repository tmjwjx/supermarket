import { NavLink, Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from './pages/LoginPage.tsx'
import RegisterPage from './pages/RegisterPage.tsx'
import UserPage from './pages/UserPage.tsx'

export default function App() {
  return (
    <>
      <nav>
        <NavLink to="/register">注册</NavLink>
        <NavLink to="/login">登录</NavLink>
        <NavLink to="/users">查资料</NavLink>
      </nav>
      <main>
        <Routes>
          <Route path="/" element={<Navigate to="/login" replace />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/users" element={<UserPage />} />
        </Routes>
      </main>
    </>
  )
}
