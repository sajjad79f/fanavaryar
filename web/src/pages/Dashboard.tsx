import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { StatsResponse } from '../api/types'
import { PageHeader, StatusBadge, EmptyState, Alert } from '../components/ui'
import { formatJalaliDateTime, formatJalali } from '../jalali'
import { actionLabel } from '../api/types'

export default function Dashboard() {
  const [data, setData] = useState<StatsResponse | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<StatsResponse>('/stats')
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [])

  if (error) return <Alert kind="error">{error}</Alert>
  if (!data) return <div className="page-loading">در حال بارگذاری...</div>

  const s = data.stats
  return (
    <>
      <PageHeader title="داشبورد" subtitle="نمای کلی فعالیت‌های واحد IT" />
      <div className="stat-grid">
        <div className="stat-card">
          <div className="stat-value">{s.assets_total}</div>
          <div className="stat-label">کل تجهیزات</div>
        </div>
        <div className="stat-card highlight">
          <div className="stat-value">{s.assets_active}</div>
          <div className="stat-label">تجهیزات فعال</div>
        </div>
        <div className="stat-card warning">
          <div className="stat-value">{s.reports_pending}</div>
          <div className="stat-label">گزارش در انتظار تایید</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{s.reports_month}</div>
          <div className="stat-label">گزارش ۳۰ روز اخیر</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{s.forms_active}</div>
          <div className="stat-label">فرم فعال</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{s.users_active}</div>
          <div className="stat-label">کاربر فعال</div>
        </div>
      </div>

      <div className="two-panels">
        <section className="card">
          <div className="card-head">
            <h2>گزارش‌های در انتظار تایید</h2>
            <Link to="/reports" className="link">
              همه گزارش‌ها
            </Link>
          </div>
          {data.pending_reports.length === 0 ? (
            <EmptyState text="گزارش در انتظار تاییدی وجود ندارد" />
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>کد</th>
                  <th>عنوان</th>
                  <th>تاریخ</th>
                  <th>ثبت‌کننده</th>
                </tr>
              </thead>
              <tbody>
                {data.pending_reports.map((r) => (
                  <tr key={r.id}>
                    <td>
                      <Link to={`/reports/${r.id}`} className="link mono">
                        {r.code ?? '—'}
                      </Link>
                    </td>
                    <td>
                      <Link to={`/reports/${r.id}`} className="link">
                        {r.title}
                      </Link>
                    </td>
                    <td>{formatJalali(r.report_date)}</td>
                    <td>{r.created_by_name ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>

        <section className="card">
          <div className="card-head">
            <h2>آخرین رویدادها</h2>
            <Link to="/audit" className="link">
              دفتر رویدادها
            </Link>
          </div>
          <ul className="audit-list">
            {data.recent_audit.map((a) => (
              <li key={a.id}>
                <span className="audit-action">{actionLabel(a.action)}</span>
                <span className="muted">{a.username ?? '—'}</span>
                <span className="muted small">{formatJalaliDateTime(a.created_at)}</span>
              </li>
            ))}
            {data.recent_audit.length === 0 && <EmptyState text="رویدادی ثبت نشده است" />}
          </ul>
        </section>
      </div>
    </>
  )
}
