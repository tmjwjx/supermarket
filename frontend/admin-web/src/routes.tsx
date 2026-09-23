import { Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom'
import AdminUsersPage from './pages/AdminUsersPage.tsx'
import AttributesPage from './pages/AttributesPage.tsx'
import AuditPage from './pages/AuditPage.tsx'
import BrandsPage from './pages/BrandsPage.tsx'
import CategoriesPage from './pages/CategoriesPage.tsx'
import LoginPage from './pages/LoginPage.tsx'
import LogsPage from './pages/LogsPage.tsx'
import OrdersPage from './pages/OrdersPage.tsx'
import ProductEditPage from './pages/ProductEditPage.tsx'
import ProductsPage from './pages/ProductsPage.tsx'
import ReconcilePage from './pages/ReconcilePage.tsx'
import RecyclePage from './pages/RecyclePage.tsx'
import ShipPage from './pages/ShipPage.tsx'
import SlotsPage from './pages/SlotsPage.tsx'
import StockPage from './pages/StockPage.tsx'
import { loadToken } from './session/index.ts'

export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route path="/" element={<Navigate to="/products" replace />} />
        <Route path="/products" element={<ProductsPage />} />
        <Route path="/products/:id" element={<ProductEditPage />} />
        <Route path="/brands" element={<BrandsPage />} />
        <Route path="/categories" element={<CategoriesPage />} />
        <Route path="/attributes" element={<AttributesPage />} />
        <Route path="/audits" element={<AuditPage />} />
        <Route path="/recycle" element={<RecyclePage />} />
        <Route path="/logs" element={<LogsPage />} />
        <Route path="/slots" element={<SlotsPage />} />
        <Route path="/stocks" element={<StockPage />} />
        <Route path="/orders" element={<OrdersPage />} />
        <Route path="/reconcile" element={<ReconcilePage />} />
        <Route path="/ship" element={<ShipPage />} />
        <Route path="/admin-users" element={<AdminUsersPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/products" replace />} />
    </Routes>
  )
}

function RequireAuth() {
  const location = useLocation()
  if (!loadToken()) {
    const from = encodeURIComponent(`${location.pathname}${location.search}`)
    return <Navigate to={`/login?from=${from}`} replace />
  }
  return <Outlet />
}
