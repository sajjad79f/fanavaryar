import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { api } from '../api/client'
import type { UserInfo } from '../api/types'

interface AuthContextValue {
  user: UserInfo | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue>({
  user: null,
  loading: true,
  login: async () => {},
  logout: async () => {},
})

/**
 * نگاشت منطقی قدیمی UI به کدهای مجوز ریزدانهٔ سرور (بخش ۳ سند دسترسی).
 * هر قابلیت UI با داشتن حداقل یکی از کدهای فهرست فعال می‌شود.
 * مرجع نهایی همیشه سمت سرور است؛ اینجا فقط برای مخفی‌سازی امکانات است.
 */
const LOGICAL_PERMS: Record<string, string[]> = {
  manage_assets: ['assets.create', 'assets.edit'],
  approve_reports: ['reports.approve'],
  manage_forms: ['forms.template_create', 'forms.template_edit', 'forms.publish', 'forms.disable'],
  manage_locations: ['locations.create', 'locations.edit', 'locations.delete'],
  manage_users: ['users.view'],
  view_audit: ['audit.view'],
  manage_user_perms: ['users.edit_permissions'],
  create_report: ['reports.create'],
}

/** بررسی مجوز منطقی قدیمی — امضای قبلی حفظ شده تا صفحات تغییر نکنند */
export function can(user: UserInfo | null, perm: string): boolean {
  if (!user) return false

  if (user.is_superadmin || user.role === 'admin') return true

  const permissions = Array.isArray(user.permissions)
    ? user.permissions
    : []

  return (LOGICAL_PERMS[perm] ?? []).some((code) =>
    permissions.includes(code)
  )
}

export function hasPerm(user: UserInfo | null, code: string): boolean {
  if (!user) return false

  if (user.is_superadmin || user.role === 'admin') return true

  const permissions = Array.isArray(user.permissions)
    ? user.permissions
    : []

  return permissions.includes(code)
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api
      .get<UserInfo>('/auth/me')
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    const handler = () => setUser(null)
    window.addEventListener('itr:unauthorized', handler)
    return () => window.removeEventListener('itr:unauthorized', handler)
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const me = await api.post<UserInfo>('/auth/login', { username, password })
    setUser(me)
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.post('/auth/logout')
    } catch {
      // بی‌اهمیت
    }
    setUser(null)
  }, [])

  return (
    <AuthContext.Provider value={{ user, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  return useContext(AuthContext)
}
