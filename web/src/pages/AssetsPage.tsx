import { useCallback, useEffect, useState } from 'react'
import { api, statusTitles } from '../api/client'
import type { Asset } from '../api/client'
import { useAuth } from '../auth/AuthContext'

export default function AssetsPage() {
  const { can } = useAuth()
  const [items, setItems] = useState<Asset[]>([])
  const [error, setError] = useState('')

  const reload = useCallback(() => {
    api
      .get<Asset[]>('/api/assets')
      .then(setItems)
      .catch((e) => setError(e.message))
  }, [])

  useEffect(reload, [reload])

  return (
    <div>
      <h1>تجهیزات</h1>
      {error && <div className="error">{error}</div>}
      <table className="table">
        <thead>
          <tr>
            <th>شناسه</th>
            <th>نام</th>
            <th>دسته</th>
            <th>وضعیت</th>
          </tr>
        </thead>
        <tbody>
          {items.map((a) => (
            <tr key={a.id}>
              <td dir="ltr">{a.assetTag}</td>
              <td>{a.name}</td>
              <td>{a.category}</td>
              <td>{statusTitles[a.status]}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {items.length === 0 && <p className="muted">تجهیزی در محدوده اطلاعاتی شما ثبت نشده است.</p>}
      {can('assets', 'create') && (
        <p className="muted">
          فرم کامل ایجاد/ویرایش تجهیز و جابه‌جایی در توسعه نهایی این صفحه تکمیل می‌شود.
        </p>
      )}
    </div>
  )
}
