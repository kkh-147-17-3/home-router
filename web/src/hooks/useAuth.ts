import { useState, useCallback } from 'react'
import { login as apiLogin } from '../api/client'

export interface AuthState {
  authenticated: boolean
  loading: boolean
  error: string | null
  login: (password: string) => Promise<void>
  logout: () => void
}

export function useAuth(): AuthState {
  const [authenticated, setAuthenticated] = useState(() => {
    return document.cookie.includes('session=') || localStorage.getItem('auth_disabled') === 'true'
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const login = useCallback(async (password: string) => {
    setLoading(true)
    setError(null)
    try {
      const res = await apiLogin(password)
      if (res.data.message === 'auth disabled') {
        localStorage.setItem('auth_disabled', 'true')
      }
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
