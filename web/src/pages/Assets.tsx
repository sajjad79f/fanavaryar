import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Asset, Paged } from '../api/types'
import { CATEGORIES, ASSET_STATUSES, labelOf } from '../api/types'
import { PageHeader, Alert, Pagination, StatusBadge } from '../components/ui'
import { AssetFormModal, useLocationOptions } from '../components/AssetForm'
import { can, useAuth } from '../context/AuthContext'

export default function Assets() {
  const { user } = useAuth()
  const canManage = can(user, 'manage_assets')

  const [q, setQ] = useState('')
  const [status, setStatus] = useState('')
  const [category, setCategory] = useState('')
  const [locationId, setLocationId] = useState('')
  const [page, setPage] = useState(1)
  const [data, setData] = useState<Paged<Asset> | null>(null)
  const [error, setError] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const locationOptions = useLocationOptions()

  const load = useCallback(() => {
    const params = new URLSearchParams({ page: String(page), per_page: '20' })
    if (q.trim()) params.set('q', q.trim())
    if (status) params.set('status', status)
    if (category) params.set('category', category)
    if (locationId) params.set('location_id', locationId)
    api
      .get<Paged<Asset>>(`/assets?${params}`)
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [q, status, category, locationId, page])

  useEffect(() => {
    load()
  }, [load])

  return (
    <>
      <PageHeader
        title="تجهیزات"
        subtitle="پرونده تجهیزات سرور، رایانه و شبکه"
        actions={
          canManage && (
            <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
              ثبت تجهیز جدید
            </button>
          )
        }
      />
      <div className="filter-bar">
        <input
          className="input grow"
          placeholder="جست‌وجو: شناسه، نام، سریال، مسئول..."
          value={q}
          onChange={(e) => {
            setQ(e.target.value)
            setPage(1)
          }}
        />
        <select className="input" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
          <option value="">همه وضعیت‌ها</option>
          {ASSET_STATUSES.map((s) => (
            <option key={s.value} value={s.value}>{s.label}</option>
          ))}
        </select>
        <select className="input" value={category} onChange={(e) => { setCategory(e.target.value); setPage(1) }}>
          <option value="">همه دسته‌ها</option>
          {CATEGORIES.map((c) => (
            <option key={c.value} value={c.value}>{c.label}</option>
          ))}
        </select>
        <select className="input" value={locationId} onChange={(e) => { setLocationId(e.target.value); setPage(1) }}>
          <option value="">همه مکان‌ها</option>
          {locationOptions.map((o) => (
            <option key={o.id} value={o.id}>{'— '.repeat(o.depth)}{o.name}</option>
          ))}
        </select>
      </div>

      {error && <Alert kind="error">{error}</Alert>}
      {data && (
        <>
          <table className="table card">
            <thead>
              <tr>
                <th>شناسه</th>
                <th>نام</th>
                <th>دسته</th>
                <th>وضعیت</th>
                <th>مکان</th>
                <th>مسئول</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((a) => (
                <tr key={a.id}>
                  <td>
                    <Link className="link mono" to={`/assets/${a.id}`}>{a.asset_tag}</Link>
                  </td>
                  <td>
                    <Link className="link" to={`/assets/${a.id}`}>{a.name}</Link>
                  </td>
                  <td>{labelOf(CATEGORIES, a.category)}</td>
                  <td><StatusBadge status={a.status} /></td>
                  <td>{a.location_name ?? '—'}</td>
                  <td>{a.responsible_person ?? '—'}</td>
                </tr>
              ))}
              {data.items.length === 0 && (
                <tr>
                  <td colSpan={6}><span className="muted">تجهیزی یافت نشد</span></td>
                </tr>
              )}
            </tbody>
          </table>
          <Pagination page={data.page} perPage={data.per_page} total={data.total} onChange={setPage} />
        </>
      )}

      {showCreate && (
        <AssetFormModal
          locations={locationOptions}
          onClose={() => setShowCreate(false)}
          onSaved={() => {
            setShowCreate(false)
            load()
          }}
        />
      )}
    </>
  )
}
