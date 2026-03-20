import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './hooks/useAuth'
import AppLayout from './components/layout/AppLayout'
import ProtectedRoute from './components/auth/ProtectedRoute'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import DHCPPage from './pages/DHCPPage'
import DNSPage from './pages/DNSPage'
import PortForwardPage from './pages/PortForwardPage'
import NetworkPage from './pages/NetworkPage'
import SystemPage from './pages/SystemPage'
import LogsPage from './pages/LogsPage'

export default function App() {
  const auth = useAuth()

  return (
    <Routes>
      <Route path="/login" element={<LoginPage auth={auth} />} />
      <Route element={<ProtectedRoute auth={auth} />}>
        <Route element={<AppLayout />}>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/dhcp" element={<DHCPPage />} />
          <Route path="/dns" element={<DNSPage />} />
          <Route path="/port-forward" element={<PortForwardPage />} />
          <Route path="/network" element={<NetworkPage />} />
          <Route path="/system" element={<SystemPage />} />
          <Route path="/logs" element={<LogsPage />} />
        </Route>
      </Route>
    </Routes>
  )
}
