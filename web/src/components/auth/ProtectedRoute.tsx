import { Navigate, Outlet } from 'react-router-dom'
import { Spin } from 'antd'
import type { AuthState } from '../../hooks/useAuth'

export default function ProtectedRoute({ auth }: { auth: AuthState }) {
  if (auth.loading) {
    return <div className="flex items-center justify-center min-h-screen"><Spin size="large" /></div>
  }
  if (!auth.authenticated) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}
