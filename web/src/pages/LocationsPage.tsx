import { useCallback, useEffect, FormEvent, useState } from 'react'
import { api, kindTitles } from '../api/client'
import type { Location, LocationNode } from '../api/client'
import { useAuth } from '../auth/AuthContext'

export default function LocationsPage() {
  const { can } = useAuth()
  const [items, setItems] = useState<Location[]>([])
  const [error, setError] = useState('')
  const [name, setName] = useState('')
  const [kind, setKind] = useState<Location['kind']>('building')
  const [parentId, setParentId] = useState<string>('')

  const reload = useCallback(() => {
    api
      .get<Location[]>('/api/locations')
      .then(setItems)
      .catch((e) => setError(e.message))
  }, [])

  useEffect(reload, [reload])

  async function create(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.post('/api/locations', {
        kind,
        name,
        parentId: parentId ? Number(parentId) : null,
      })
      setName('')
      setParentId('')
      reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'ثبت ناموفق بود')
    }
  }

  async function remove(loc: Location) {
    if (!confirm(`مکان «${loc.name}» حذف شود؟`)) return
    try {
      await api.delete(`/api/locations/${loc.id}`)
      reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'حذف ناموفق بود')
    }
  }

  const candidates = items.filter((i) => i.kind !== 'rack')

  return (
    <div>
      <h1>مکان‌ها</h1>
      {error && <div className="error">{error}</div>}

      {can('locations', 'create') && (
        <form className="inline-form" onSubmit={create}>
          <select value={kind} onChange={(e) => setKind(e.target.value as Location['kind'])}>
            <option value="building">ساختمان</option>
            <option value="floor">طبقه</option>
            <option value="room">اتاق</option>
            <option value="rack">رک</option>
          </select>
          <select value={parentId} onChange={(e) => setParentId(e.target.value)}>
            <option value="">بدون والد (فقط ساختمان)</option>
            {candidates.map((c) => (
              <option key={c.id} value={c.id}>
                {kindTitles[c.kind]}: {c.name}
              </option>
            ))}
          </select>
          <input
            placeholder="نام مکان"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
          <button type="submit">افزودن</button>
        </form>
      )}

      <table className="table">
        <thead>
          <tr>
            <th>نوع</th>
            <th>نام</th>
            <th>کد</th>
            <th>والد</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {items.map((loc) => (
            <tr key={loc.id}>
              <td>{kindTitles[loc.kind]}</td>
              <td>{loc.name}</td>
              <td>{loc.code ?? '—'}</td>
              <td>
                {loc.parentId
                  ? items.find((i) => i.id === loc.parentId)?.name ?? loc.parentId
                  : '—'}
              </td>
              <td>
                {can('locations', 'delete') && (
                  <button onClick={() => void remove(loc)}>حذف</button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {items.length === 0 && <p className="muted">هنوز مکانی ثبت نشده است.</p>}
    </div>
  )
}

// نمودار درختی برای نمایش آینده — فاز ۲ در صفحه اختصاصی به کار می‌رود
export function toTree(items: Location[]): LocationNode[] {
  return items.map((i) => ({ ...i, children: [] }))
}
