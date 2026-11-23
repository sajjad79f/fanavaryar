import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { AuthUser } from '../api/client'
import { useAuth } from '../auth/AuthContext'

// صفحه شروع مدیریت کاربران؛ ویرایش کامل مجوزها در توسعه نهایی تکمیل می‌شود.
export default function UsersPage() {
  const { can } = useAuth()
  const [items, setItems] = useState<AuthUser[]>([])
  const [error, setError] = useState('')

  const reload = useCallback(() => {
    api
      .get<AuthUser[]>('/api/users')
      .then(setItems)
      .catch((e) => setError(e.message))
  }, [])

  useEffect(reload, [reload])

  return (
    <div>
      <h1>کاربران</h1>
      {error && <div className="error">{error}</div>}
      <table className="table">
        <thead>
          <tr>
            <th>نام کاربری</th>
            <th>نام و نام خانوادگی</th>
            <th>سمت سازمانی</th>
            <th>وضعیت</th>
            <th>محدوده اطلاعاتی</th>
          </tr>
        </thead>
        <tbody>
          {items.map((u) => (
            <tr key={u.id}>
              <td dir="ltr">{u.username}</td>
              <td>{u.fullName}</td>
              <td>{u.position ?? '—'}</td>
              <td>{(u as unknown as { isActive?: boolean }).isActive === false ? 'غیرفعال' : 'فعال'}</td>
              <td>{u.scopeType}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {items.length === 0 && <p className="muted">کاربری یافت نشد.</p>}
      {can('permissions', 'grant') && (
        <p className="muted">
          مدیریت مجوزهای ریزدانه (تخصیص به‌تفکیک ماژول و عملیات، الگوهای دسترسی
          و محدوده اطلاعاتی) از این صفحه در توسعه نهایی تکمیل می‌شود.
        </p>
      )}
    </div>
  )
}
