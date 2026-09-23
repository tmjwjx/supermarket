import { NavLink } from 'react-router-dom'
import AppRoutes from './routes.tsx'

export default function App() {
  return (
    <>
      <nav>
        <NavLink to="/" end>
          首页
        </NavLink>
      </nav>
      <main>
        <AppRoutes />
      </main>
    </>
  )
}
