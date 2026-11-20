import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth } from './context/AuthContext'
import Layout from './components/Layout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Assets from './pages/Assets'
import AssetDetail from './pages/AssetDetail'
import Locations from './pages/Locations'
import Forms from './pages/Forms'
import FormDetail from './pages/FormDetail'
import NewReport from './pages/NewReport'
import Reports from './pages/Reports'
import ReportDetail from './pages/ReportDetail'
import Users from './pages/Users'
import Audit from './pages/Audit'

function Protected({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) {
    return <div className="page-loading">در حال بارگذاری...</div>
  }
  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <Protected>
            <Layout />
          </Protected>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="assets" element={<Assets />} />
        <Route path="assets/:id" element={<AssetDetail />} />
        <Route path="locations" element={<Locations />} />
        <Route path="forms" element={<Forms />} />
        <Route path="forms/:id" element={<FormDetail />} />
        <Route path="reports" element={<Reports />} />
        <Route path="reports/new" element={<NewReport />} />
        <Route path="reports/:id" element={<ReportDetail />} />
        <Route path="users" element={<Users />} />
        <Route path="audit" element={<Audit />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
