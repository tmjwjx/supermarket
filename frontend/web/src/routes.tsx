import { Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from './pages/LoginPage.tsx'
import ProfilePage from './pages/ProfilePage.tsx'
import RegisterPage from './pages/RegisterPage.tsx'

export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/me" element={<ProfilePage />} />
    </Routes>
  )
}
