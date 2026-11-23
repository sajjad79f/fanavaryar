import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { AssetDetail } from '../api/types'
import { CATEGORIES, labelOf } from '../api/types'
import { PageHeader, Alert, Modal, StatusBadge } from '../components/ui'
import { AssetFormModal, useLocationOptions } from '../components/AssetForm'
import { displayValue } from '../components/DynamicForm'
import { can, useAuth } from '../context/AuthContext'
import { formatJalali, formatJalaliDateTime } from '../jalali'

export default function AssetDetail() {
  const { id } = useParams()
  const { user } = useAuth()
  const canManage = can(user, 'manage_assets')

  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [error, setError] = useState('')
  const [showEdit, setShowEdit] = useState(false)
  const [showMove, setShowMove] = useState(false)
  const locationOptions = useLocationOptions()

  const load = useCallback(() => {
    api
      .get<AssetDetail>(`/assets/${id}`)
      .then(setAsset)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [id])

  useEffect(() => {
    load()
  }, [load])

  if (error) return <Alert kind="error">{error}</Alert>
  if (!asset) return <div className="page-loading">در حال بارگذاری...</div>

  return (
    <>
      <PageHeader
        title={`${asset.name} — ${asset.asset_tag}`}
        subtitle={labelOf(CATEGORIES, asset.category)}
        actions={
          canManage && (
            <>
              <button className="btn btn-secondary" onClick={() => setShowMove(true)}>جابه‌جایی</button>
              <button className="btn btn-primary" onClick={() => setShowEdit(true)}>ویرایش</button>
            </>
          )
        }
      />
      <div className="detail-grid">
        <div className="detail-item"><span className="detail-label">وضعیت</span><StatusBadge status={asset.status} /></div>
        <div className="detail-item"><span className="detail-label">مکان فعلی</span>{asset.location_name ?? '—'}</div>
        <div className="detail-item"><span className="detail-label">شماره سریال</span><span dir="ltr">{asset.serial_number ?? '—'}</span></div>
        <div className="detail-item"><span className="detail-label">سازنده / مدل</span>{asset.manufacturer ?? '—'} {asset.model ?? ''}</div>
        <div className="detail-item"><span className="detail-label">مسئول</span>{asset.responsible_person ?? '—'}</div>
        <div className="detail-item"><span className="detail-label">تاریخ ثبت</span>{formatJalaliDateTime(asset.created_at)}</div>
      </div>
      {asset.notes && <div className="card notes">توضیحات: {asset.notes}</div>}

      {asset.spec_form_version_id && (
        <section className="card">
          <div className="card-head">
            <h2>مشخصات سخت‌افزاری</h2>
            <span className="muted small">
              فرم «{asset.spec_template_name}» — نسخه {asset.spec_version_number}
            </span>
          </div>
          <div className="detail-grid">
            {asset.spec_fields.map((f) => (
              <div className="detail-item" key={f.key}>
                <span className="detail-label">{f.label}</span>
                {displayValue(f, asset.spec_data[f.key])}
              </div>
            ))}
          </div>
        </section>
      )}

      <section className="card">
        <div className="card-head"><h2>تاریخچه جابه‌جایی</h2></div>
        {asset.movements.length === 0 ? (
          <p className="muted">جابه‌جایی‌ای ثبت نشده است</p>
        ) : (
          <table className="table compact">
            <thead>
              <tr>
                <th>زمان</th>
                <th>از مکان</th>
                <th>به مکان</th>
                <th>دلیل</th>
                <th>عملیات‌کننده</th>
              </tr>
            </thead>
            <tbody>
              {asset.movements.map((m) => (
                <tr key={m.id}>
                  <td>{formatJalaliDateTime(m.created_at)}</td>
                  <td>{m.from_location_name ?? '—'}</td>
                  <td>{m.to_location_name ?? '—'}</td>
                  <td>{m.reason ?? '—'}</td>
                  <td>{m.moved_by_name ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section className="card">
        <div className="card-head"><h2>گزارش‌های مرتبط</h2></div>
        {asset.reports.length === 0 ? (
          <p className="muted">گزارشی به این تجهیز متصل نیست</p>
        ) : (
          <table className="table compact">
            <thead>
              <tr>
                <th>کد</th>
                <th>عنوان</th>
                <th>تاریخ</th>
                <th>وضعیت</th>
              </tr>
            </thead>
            <tbody>
              {asset.reports.map((r) => (
                <tr key={r.id}>
                  <td>
                    <Link className="link mono" to={`/reports/${r.id}`}>{r.code ?? '—'}</Link>
                  </td>
                  <td>
                    <Link className="link" to={`/reports/${r.id}`}>{r.title}</Link>
                  </td>
                  <td>{formatJalali(r.report_date)}</td>
                  <td><StatusBadge status={r.status} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {showEdit && (
        <AssetFormModal
          asset={asset}
          locations={locationOptions}
          onClose={() => setShowEdit(false)}
          onSaved={() => {
            setShowEdit(false)
            load()
          }}
        />
      )}
      {showMove && (
        <MoveModal
          asset={asset}
          locations={locationOptions}
          onClose={() => setShowMove(false)}
          onSaved={() => {
            setShowMove(false)
            load()
          }}
        />
      )}
    </>
  )
}

function MoveModal({
  asset,
  locations,
  onClose,
  onSaved,
}: {
  asset: AssetDetail
  locations: { id: number; name: string; depth: number }[]
  onClose: () => void
  onSaved: () => void
}) {
  const [toLocation, setToLocation] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!toLocation) {
      setError('مکان مقصد را انتخاب کنید')
      return
    }
    setBusy(true)
    try {
      await api.post(`/assets/${asset.id}/move`, {
        to_location_id: Number(toLocation),
        reason: reason.trim() || undefined,
      })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={`جابه‌جایی ${asset.asset_tag}`} onClose={onClose}>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="form-row">
          <label>مکان فعلی</label>
          <input className="input" value={asset.location_name ?? '—'} disabled />
        </div>
        <div className="form-row">
          <label>مکان مقصد *</label>
          <select className="input" value={toLocation} onChange={(e) => setToLocation(e.target.value)}>
            <option value="">— انتخاب مکان —</option>
            {locations
              .filter((o) => o.id !== asset.current_location_id)
              .map((o) => (
                <option key={o.id} value={o.id}>{'— '.repeat(o.depth)}{o.name}</option>
              ))}
          </select>
        </div>
        <div className="form-row">
          <label>دلیل جابه‌جایی</label>
          <input className="input" value={reason} onChange={(e) => setReason(e.target.value)} />
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? '...' : 'ثبت جابه‌جایی'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
