import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { AuditItem, Paged } from '../api/types'
import { actionLabel } from '../api/types'
import { PageHeader, Alert, Pagination, EmptyState } from '../components/ui'
import { formatJalaliDateTime } from '../jalali'

export default function Audit() {
  const [q, setQ] = useState('')
  const [page, setPage] = useState(1)
  const [data, setData] = useState<Paged<AuditItem> | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    const params = new URLSearchParams({ page: String(page), per_page: '25' })
    if (q.trim()) params.set('q', q.trim())
    api
      .get<Paged<AuditItem>>(`/audit?${params}`)
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [q, page])

  useEffect(() => {
    load()
  }, [load])

  return (
    <>
      <PageHeader
        title="دفتر رویدادها"
        subtitle="سوابق تغییرناپذیر رویدادهای سامانه (فقط الحاقی)"
      />
      <div className="filter-bar">
        <input
          className="input grow"
          placeholder="جست‌وجو: نوع رویداد، کاربر..."
          value={q}
          onChange={(e) => {
            setQ(e.target.value)
            setPage(1)
          }}
        />
      </div>
      {error && <Alert kind="error">{error}</Alert>}
      {data && (
        <>
          <table className="table card">
            <thead>
              <tr>
                <th>زمان</th>
                <th>کاربر</th>
                <th>رویداد</th>
                <th>موجودیت</th>
                <th>جزئیات</th>
                <th>IP</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((a) => (
                <tr key={a.id}>
                  <td className="nowrap">{formatJalaliDateTime(a.created_at)}</td>
                  <td>{a.full_name ?? a.username ?? '—'}</td>
                  <td>
                    <span className={`badge badge-muted act-${a.action.replace('.', '-')}`}>
                      {actionLabel(a.action)}
                    </span>
                  </td>
                  <td>
                    {a.entity_type ? `${entityLabel(a.entity_type)} #${a.entity_id ?? ''}` : '—'}
                  </td>
                  <td className="mono small ellipsis" dir="ltr">
                    {a.details ? JSON.stringify(a.details) : '—'}
                  </td>
                  <td dir="ltr">{a.ip ?? '—'}</td>
                </tr>
              ))}
              {data.items.length === 0 && (
                <tr>
                  <td colSpan={6}><EmptyState text="رویدادی یافت نشد" /></td>
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

function entityLabel(t: string): string {
  const map: Record<string, string> = {
    user: 'کاربر',
    asset: 'تجهیز',
    report: 'گزارش',
    form_template: 'فرم',
    location: 'مکان',
  }
  return map[t] ?? t
}
