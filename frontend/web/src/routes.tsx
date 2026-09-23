import { Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom'
import AddressesPage from './pages/AddressesPage.tsx'
import BrowsePage from './pages/BrowsePage.tsx'
import CartPage from './pages/CartPage.tsx'
import CategoryPage from './pages/CategoryPage.tsx'
import FavoritesPage from './pages/FavoritesPage.tsx'
import HomePage from './pages/HomePage.tsx'
import LoginPage from './pages/LoginPage.tsx'
import MessagesPage from './pages/MessagesPage.tsx'
import OrderDetailPage from './pages/OrderDetailPage.tsx'
import OrdersPage from './pages/OrdersPage.tsx'
import PayPage from './pages/PayPage.tsx'
import ProductPage from './pages/ProductPage.tsx'
import ProfilePage from './pages/ProfilePage.tsx'
import RegisterPage from './pages/RegisterPage.tsx'
import ReviewPage from './pages/ReviewPage.tsx'
import SearchPage from './pages/SearchPage.tsx'
import { loadToken } from './session/index.ts'

export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/products/:id" element={<ProductPage />} />
      <Route path="/category/:id" element={<CategoryPage />} />
      <Route path="/search" element={<SearchPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/login" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route path="/cart" element={<CartPage />} />
        <Route path="/orders" element={<OrdersPage />} />
        <Route path="/orders/:id" element={<OrderDetailPage />} />
        <Route path="/orders/:id/review/:itemId" element={<ReviewPage />} />
        <Route path="/addresses" element={<AddressesPage />} />
        <Route path="/favorites" element={<FavoritesPage />} />
        <Route path="/history" element={<BrowsePage />} />
        <Route path="/messages" element={<MessagesPage />} />
        <Route path="/pay/:id" element={<PayPage />} />
        <Route path="/me" element={<ProfilePage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
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
