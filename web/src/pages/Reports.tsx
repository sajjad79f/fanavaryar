import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { ReportListItem, Paged } from '../api/types'
import { REPORT_STATUSES } from '../api/types'
import { PageHeader, Alert, Pagination, StatusBadge } from '../components/ui'
import { formatJalali, formatJalaliDateTime } from '../jalali'
import { useAuth, hasPerm } from '../context/AuthContext'

export default function Reports() {
  const { user } = useAuth()
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [data, setData] = useState<Paged<ReportListItem> | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    const params = new URLSearchParams({ page: String(page), per_page: '20' })
    if (q.trim()) params.set('q', q.trim())
    if (status) params.set('status', status)
    api
      .get<Paged<ReportListItem>>(`/reports?${params}`)
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [q, status, page])

  useEffect(() => {
    load()
  }, [load])

  return (
    <>
      <PageHeader
        title="گزارش‌ها"
        subtitle="سوابق فعالیت‌ها؛ هر گزارش به تجهیزات مرتبط متصل است"
        actions={
          hasPerm(user, 'reports.create') && (
            <Link className="btn btn-primary" to="/reports/new">
              ثبت گزارش جدید
            </Link>
          )
        }
      />
      <div className="filter-bar">
        <input
          className="input grow"
          placeholder="جست‌وجو: کد یا عنوان گزارش..."
          value={q}
          onChange={(e) => {
            setQ(e.target.value)
            setPage(1)
          }}
        />
        <select className="input" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
          <option value="">همه وضعیت‌ها</option>
          {REPORT_STATUSES.map((s) => (
            <option key={s.value} value={s.value}>{s.label}</option>
          ))}
        </select>
      </div>

      {error && <Alert kind="error">{error}</Alert>}
      {data && (
        <>
          <table className="table card">
            <thead>
              <tr>
                <th>کد</th>
                <th>عنوان</th>
                <th>تاریخ گزارش</th>
                <th>ثبت‌کننده</th>
                <th>تاریخ ثبت</th>
                <th>وضعیت</th>
                <th>بررسی‌کننده</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((r) => (
                <tr key={r.id}>
                  <td>
                    <Link className="link mono" to={`/reports/${r.id}`}>{r.code ?? '—'}</Link>
                  </td>
                  <td>
                    <Link className="link" to={`/reports/${r.id}`}>{r.title}</Link>
                  </td>
                  <td>{formatJalali(r.report_date)}</td>
                  <td>{r.created_by_name ?? '—'}</td>
                  <td>{formatJalaliDateTime(r.created_at)}</td>
                  <td><StatusBadge status={r.status} /></td>
                  <td>{r.approved_by_name ?? '—'}</td>
                </tr>
              ))}
              {data.items.length === 0 && (
                <tr>
                  <td colSpan={7}><span className="muted">گزارشی یافت نشد</span></td>
                </tr>
              )}
            </tbody>
          </table>
          <Pagination page={data.page} perPage={data.per_page} total={data.total} onChange={setPage} />
        </>
      )}
    </>
  )
}
