import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { api, hasPermission } from '../api/client'
import type { AuthUser } from '../api/client'

interface AuthState {
  user: AuthUser | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  can: (module: string, action: string) => boolean
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get<AuthUser>('/api/auth/me')
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    await api.post('/api/auth/login', { username, password })
    const me = await api.get<AuthUser>('/api/auth/me')
    setUser(me)
  }, [])

  const logout = useCallback(async () => {
    await api.post('/api/auth/logout')
    setUser(null)
  }, [])

  const can = useCallback(
    (module: string, action: string) => hasPermission(user, module, action),
    [user],
  )

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, can }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
