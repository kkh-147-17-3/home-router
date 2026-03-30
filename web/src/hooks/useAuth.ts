import { useState, useCallback, useEffect } from 'react'
import { login as apiLogin, checkAuth } from '../api/client'

export interface AuthState {
  authenticated: boolean
  loading: boolean
  error: string | null
  login: (password: string) => Promise<void>
  logout: () => void
}

export function useAuth(): AuthState {
  const [authenticated, setAuthenticated] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // 초기 인증 상태 확인
  useEffect(() => {
    checkAuth()
      .then((res) => {
        setAuthenticated(res.authenticated)
        if (res.auth_disabled) {
          localStorage.setItem('auth_disabled', 'true')
        }
      })
      .catch(() => setAuthenticated(false))
      .finally(() => setLoading(false))
  }, [])

  const login = useCallback(async (password: string) => {
    setLoading(true)
    setError(null)
    try {
      await apiLogin(password)
      setAuthenticated(true)
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Login failed'
      setError(msg)
      throw e
    } finally {
      setLoading(false)
    }
  }, [])

  const logout = useCallback(() => {
    document.cookie = 'session=; Max-Age=0; path=/'
    localStorage.removeItem('auth_disabled')
    setAuthenticated(false)
  }, [])

  return { authenticated, loading, error, login, logout }
}
