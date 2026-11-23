import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { FormDetail, FormVersion, FieldDef } from '../api/types'
import { formKindLabel } from '../api/types'
import { PageHeader, Modal, Alert, StatusBadge } from '../components/ui'
import { FieldsEditor } from '../components/FieldsEditor'
import { DynamicForm } from '../components/DynamicForm'
import { hasPerm, useAuth } from '../context/AuthContext'
import { formatJalaliDateTime } from '../jalali'

export default function FormDetail() {
  const { id } = useParams()
  const { user } = useAuth()
  const canEditTemplate = hasPerm(user, 'forms.template_edit')

  const [detail, setDetail] = useState<FormDetail | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [showNewVersion, setShowNewVersion] = useState(false)
  const [selectedVersion, setSelectedVersion] = useState<FormVersion | null>(null)

  const load = useCallback(() => {
    api
      .get<FormDetail>(`/forms/${id}`)
      .then((d) => {
        setDetail(d)
        const current = d.versions.find((v) => v.id === d.current_version_id) ?? d.versions[0] ?? null
        setSelectedVersion(current)
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [id])

  useEffect(() => {
    load()
  }, [load])

  if (error) return <Alert kind="error">{error}</Alert>
  if (!detail) return <div className="page-loading">در حال بارگذاری...</div>

  async function toggleActive() {
    if (!detail) return
    try {
      await api.patch(`/forms/${detail.id}`, { is_active: !detail.is_active })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'خطا')
    }
  }

  async function toggleApproval() {
    if (!detail) return
    try {
      await api.patch(`/forms/${detail.id}`, { requires_approval: !detail.requires_approval })
      setNotice(
        detail.requires_approval
          ? 'نیاز به تایید برای این فرم غیرفعال شد؛ گزارش‌های بعدی مستقیم نهایی می‌شوند'
          : 'نیاز به تایید برای این فرم فعال شد؛ گزارش‌های بعدی به صف بررسی می‌روند',
      )
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'خطا')
    }
  }

  return (
    <>
      <PageHeader
        title={detail.name}
        subtitle={`کد: ${detail.code} — نوع: ${formKindLabel(detail.kind)}`}
        actions={
          <>
            {(hasPerm(user, 'forms.publish') || hasPerm(user, 'forms.disable')) && (
              <button
                className="btn btn-secondary"
                onClick={toggleActive}
                disabled={detail.is_active ? !hasPerm(user, 'forms.disable') : !hasPerm(user, 'forms.publish')}
                title={detail.is_active ? 'نیازمند مجوز غیرفعال‌سازی فرم' : 'نیازمند مجوز انتشار فرم'}
              >
                {detail.is_active ? 'غیرفعال‌سازی فرم' : 'فعال‌سازی فرم'}
              </button>
            )}
            {canEditTemplate && (
              <button className="btn btn-primary" onClick={() => setShowNewVersion(true)}>
                ثبت نسخه جدید
              </button>
            )}
          </>
        }
      />
      {notice && <Alert kind="success">{notice}</Alert>}
      {error && <Alert kind="error">{error}</Alert>}
      {detail.description && <p className="muted">{detail.description}</p>}

      <div className="card" style={{ padding: '0.75rem 1rem', display: 'flex', alignItems: 'center', gap: '1rem', flexWrap: 'wrap' }}>
        <span>
          <strong>سیاست تایید:</strong>{' '}
          {detail.requires_approval
            ? 'گزارش‌های این فرم نیاز به تایید مدیر دارند'
            : 'گزارش‌های این فرم در لحظه ثبت نهایی می‌شوند (بدون تایید)'}
        </span>
        {canEditTemplate && (
          <button className="btn btn-ghost small" onClick={toggleApproval}>
            {detail.requires_approval ? 'غیرفعال‌سازی نیاز به تایید' : 'فعال‌سازی نیاز به تایید'}
          </button>
        )}
      </div>

      <div className="two-panels">
        <section className="card">
          <div className="card-head">
            <h2>نسخه‌ها</h2>
            <span className="muted small">نسخه‌ها تغییرناپذیرند؛ اصلاح فرم یعنی ثبت نسخه جدید</span>
          </div>
          <table className="table compact">
            <thead>
              <tr>
                <th>نسخه</th>
                <th>تعداد فیلد</th>
                <th>ساخته‌کننده</th>
                <th>تاریخ</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {detail.versions.map((v) => (
                <tr key={v.id} className={v.id === detail.current_version_id ? 'highlight-row' : ''}>
                  <td>
                    <span className="badge badge-muted">v{v.version_number}</span>
                    {v.id === detail.current_version_id && <span className="muted small"> (جاری)</span>}
                  </td>
                  <td>{v.fields.length}</td>
                  <td>{v.created_by_name ?? '—'}</td>
                  <td>{formatJalaliDateTime(v.created_at)}</td>
                  <td>
                    <button className="btn btn-ghost small" onClick={() => setSelectedVersion(v)}>
                      نمایش
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>

        <section className="card">
          <div className="card-head">
            <h2>فیلدهای نسخه {selectedVersion ? selectedVersion.version_number : '—'}</h2>
          </div>
          {selectedVersion ? (
            <FieldList fields={selectedVersion.fields} />
          ) : (
            <p className="muted">نسخه‌ای یافت نشد</p>
          )}
        </section>
      </div>

      {selectedVersion && (
        <section className="card">
          <div className="card-head">
            <h2>پیش‌نمایش فرم</h2>
            <StatusBadge status="active" />
          </div>
          <DynamicForm
            fields={selectedVersion.fields}
            value={{}}
            onChange={() => {}}
            disabled
          />
        </section>
      )}

      {showNewVersion && (
        <NewVersionModal
          templateId={detail.id}
          baseFields={selectedVersion?.fields ?? []}
          onClose={() => setShowNewVersion(false)}
          onSaved={() => {
            setShowNewVersion(false)
            setNotice('نسخه جدید با موفقیت ثبت شد')
            load()
          }}
        />
      )}
    </>
  )
}

function FieldList({ fields }: { fields: FieldDef[] }) {
  return (
    <table className="table compact">
      <thead>
        <tr>
          <th>برچسب</th>
          <th>کلید</th>
          <th>نوع</th>
          <th>الزامی</th>
          <th>گزینه‌ها</th>
        </tr>
      </thead>
      <tbody>
        {fields.map((f) => (
          <tr key={f.key}>
            <td>{f.label}</td>
            <td dir="ltr" className="mono">{f.key}</td>
            <td>{f.type}</td>
            <td>{f.required ? 'بله' : 'خیر'}</td>
            <td>{f.type === 'select' ? (f.options ?? []).join('، ') : '—'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function NewVersionModal({
  templateId,
  baseFields,
  onClose,
  onSaved,
}: {
  templateId: number
  baseFields: FieldDef[]
  onClose: () => void
  onSaved: () => void
}) {
  const [fields, setFields] = useState<FieldDef[]>(baseFields.map((f) => ({ ...f })))
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (fields.length === 0) {
      setError('حداقل یک فیلد لازم است')
      return
    }
    setBusy(true)
    try {
      await api.post(`/forms/${templateId}/versions`, { fields })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="ثبت نسخه جدید فرم" onClose={onClose} width={680}>
      <p className="muted small">
        نسخه جاری به‌عنوان نقطه شروع کپی شده است. تغییرات فقط در نسخه جدید اعمال می‌شود و گزارش‌های قبلی دست‌نخورده می‌مانند.
      </p>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <FieldsEditor fields={fields} onChange={setFields} />
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? '...' : 'ثبت نسخه'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
