import { Navigate, Outlet } from 'react-router-dom'
import type { AuthState } from '../../hooks/useAuth'

export default function ProtectedRoute({ auth }: { auth: AuthState }) {
  if (!auth.authenticated) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}
