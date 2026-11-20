import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { ReportDetail } from '../api/types'
import { PageHeader, Alert, Modal, StatusBadge } from '../components/ui'
import { displayValue } from '../components/DynamicForm'
import { can, useAuth } from '../context/AuthContext'
import { formatJalali, formatJalaliDateTime } from '../jalali'

export default function ReportDetail() {
  const { id } = useParams()
  const { user } = useAuth()
  const canApprove = can(user, 'approve_reports')

  const [report, setReport] = useState<ReportDetail | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [showReject, setShowReject] = useState(false)
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => {
    api
      .get<ReportDetail>(`/reports/${id}`)
      .then(setReport)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [id])

  useEffect(() => {
    load()
  }, [load])

  if (error && !report) return <Alert kind="error">{error}</Alert>
  if (!report) return <div className="page-loading">در حال بارگذاری...</div>

  async function approve() {
    if (!window.confirm('تایید این گزارش؟')) return
    setBusy(true)
    setError('')
    try {
      await api.post(`/reports/${report!.id}/approve`)
      setNotice('گزارش تایید شد')
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader
        title={report.title}
        subtitle={`کد گزارش: ${report.code ?? '—'}`}
        actions={<StatusBadge status={report.status} />}
      />
      {notice && <Alert kind="success">{notice}</Alert>}
      {error && <Alert kind="error">{error}</Alert>}
      {report.status === 'rejected' && report.reject_reason && (
        <Alert kind="error">دلیل رد: {report.reject_reason}</Alert>
      )}

      <div className="detail-grid">
        <div className="detail-item"><span className="detail-label">تاریخ گزارش</span>{formatJalali(report.report_date)}</div>
        <div className="detail-item"><span className="detail-label">ثبت‌کننده</span>{report.created_by_name ?? '—'}</div>
        <div className="detail-item"><span className="detail-label">زمان ثبت</span>{formatJalaliDateTime(report.created_at)}</div>
        <div className="detail-item">
          <span className="detail-label">فرم</span>
          {report.template_name} — نسخه {report.version_number}
        </div>
        <div className="detail-item"><span className="detail-label">بررسی‌کننده</span>{report.approved_by_name ?? '—'}</div>
        <div className="detail-item"><span className="detail-label">زمان بررسی</span>{report.approved_at ? formatJalaliDateTime(report.approved_at) : '—'}</div>
      </div>

      <section className="card">
        <div className="card-head"><h2>تجهیزات مرتبط</h2></div>
        {report.assets.length === 0 ? (
          <p className="muted">تجهیزی به این گزارش متصل نشده است</p>
        ) : (
          <div className="chips">
            {report.assets.map((a) => (
              <Link key={a.id} className="chip link-chip" to={`/assets/${a.id}`}>
                {a.asset_tag} — {a.name}
              </Link>
            ))}
          </div>
        )}
      </section>

      <section className="card">
        <div className="card-head"><h2>داده‌های فرم</h2></div>
        <table className="table compact">
          <tbody>
            {report.fields.map((f) => (
              <tr key={f.key}>
                <th className="w-30">{f.label}</th>
                <td>{displayValue(f, report.form_data[f.key])}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {report.status === 'pending' && canApprove && (
        <section className="card approval-panel">
          <div className="card-head"><h2>بررسی گزارش</h2></div>
          <p className="muted">این گزارش در انتظار تایید شماست. توجه: بررسی گزارش ثبت‌شده توسط خودتان مجاز نیست.</p>
          <div className="form-actions">
            <button className="btn btn-danger-ghost" onClick={() => setShowReject(true)} disabled={busy}>
              رد گزارش
            </button>
            <button className="btn btn-primary" onClick={approve} disabled={busy}>
              تایید گزارش
            </button>
          </div>
        </section>
      )}

      {showReject && (
        <RejectModal
          reportId={report.id}
          onClose={() => setShowReject(false)}
          onSaved={() => {
            setShowReject(false)
            setNotice('گزارش رد شد')
            load()
          }}
        />
      )}
    </>
  )
}

function RejectModal({
  reportId,
  onClose,
  onSaved,
}: {
  reportId: number
  onClose: () => void
  onSaved: () => void
}) {
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (reason.trim().length < 3) {
      setError('دلیل رد گزارش الزامی است')
      return
    }
    setBusy(true)
    try {
      await api.post(`/reports/${reportId}/reject`, { reason: reason.trim() })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="رد گزارش" onClose={onClose}>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="form-row">
          <label>دلیل رد (برای ثبت‌کننده نمایش داده می‌شود) *</label>
          <textarea className="input" rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-danger" disabled={busy}>{busy ? '...' : 'ثبت رد'}</button>
        </div>
      </form>
    </Modal>
  )
}
