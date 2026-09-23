import { Navigate, Route, Routes } from 'react-router-dom'
import HomePage from './pages/HomePage.tsx'

export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
