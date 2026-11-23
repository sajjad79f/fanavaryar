import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { FormKind, FormTemplate } from '../api/types'
import { PageHeader, Modal, Alert } from '../components/ui'
import { FieldsEditor } from '../components/FieldsEditor'
import type { FieldDef } from '../api/types'
import { formKindLabel, FORM_KINDS } from '../api/types'
import { hasPerm, useAuth } from '../context/AuthContext'
import { formatJalaliDateTime } from '../jalali'

export default function Forms() {
  const { user } = useAuth()
  const canCreate = hasPerm(user, 'forms.template_create')
  const navigate = useNavigate()
  const [templates, setTemplates] = useState<FormTemplate[]>([])
  const [error, setError] = useState('')
  const [showCreate, setShowCreate] = useState(false)

  const load = useCallback(() => {
    api
      .get<FormTemplate[]>('/forms')
      .then(setTemplates)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  return (
    <>
      <PageHeader
        title="فرم‌ها"
        subtitle="قالب‌های دیجیتال فرم‌های واحد IT با نسخه‌بندی"
        actions={canCreate && <button className="btn btn-primary" onClick={() => setShowCreate(true)}>فرم جدید</button>}
      />
      {error && <Alert kind="error">{error}</Alert>}
      <table className="table card">
        <thead>
          <tr>
            <th>کد</th>
            <th>نام</th>
            <th>نوع</th>
            <th>توضیح</th>
            <th>وضعیت</th>
            <th>تاریخ ایجاد</th>
          </tr>
        </thead>
        <tbody>
          {templates.map((t) => (
            <tr key={t.id} className={!t.is_active ? 'dim' : ''}>
              <td>
                <Link className="link mono" to={`/forms/${t.id}`}>{t.code}</Link>
              </td>
              <td>
                <Link className="link" to={`/forms/${t.id}`}>{t.name}</Link>
              </td>
              <td>
                <span className="badge badge-muted">{formKindLabel(t.kind)}</span>
              </td>
              <td>{t.description ?? '—'}</td>
              <td>
                {t.is_active ? <span className="badge badge-success">فعال</span> : <span className="badge badge-muted">غیرفعال</span>}
                {!t.requires_approval && <span className="badge badge-muted">بدون نیاز به تایید</span>}
              </td>
              <td>{formatJalaliDateTime(t.created_at)}</td>
            </tr>
          ))}
          {templates.length === 0 && (
            <tr>
              <td colSpan={6}><span className="muted">هنوز فرمی تعریف نشده است</span></td>
            </tr>
          )}
        </tbody>
      </table>

      {showCreate && (
        <CreateFormModal
          onClose={() => setShowCreate(false)}
          onCreated={(id) => {
            setShowCreate(false)
            load()
            navigate(`/forms/${id}`)
          }}
        />
      )}
    </>
  )
}

function CreateFormModal({ onClose, onCreated }: { onClose: () => void; onCreated: (id: number) => void }) {
  const [code, setCode] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [kind, setKind] = useState<FormKind>('report')
  const [requiresApproval, setRequiresApproval] = useState(true)
  const [fields, setFields] = useState<FieldDef[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!name.trim() || !code.trim()) {
      setError('کد و نام فرم الزامی است')
      return
    }
    if (fields.length === 0) {
      setError('حداقل یک فیلد برای فرم لازم است')
      return
    }
    setBusy(true)
    try {
      const res = await api.post<{ id: number }>('/forms', {
        code: code.trim(),
        name: name.trim(),
        description: description.trim() || undefined,
        kind,
        requires_approval: requiresApproval,
        fields,
      })
      onCreated(res.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="فرم جدید" onClose={onClose} width={680}>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="two-cols">
          <div className="form-row">
            <label>کد فرم (انگلیسی) *</label>
            <input
              className="input"
              dir="ltr"
              placeholder="IT-Service-Form"
              value={code}
              onChange={(e) => setCode(e.target.value)}
            />
            <span className="muted small">حروف انگلیسی بزرگ/کوچک، رقم، خط تیره و زیرخط مجاز است؛ باید با حرف شروع شود</span>
          </div>
          <div className="form-row">
            <label>نام فرم *</label>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
        </div>
        <div className="two-cols">
          <div className="form-row">
            <label>نوع فرم *</label>
            <select className="input" value={kind} onChange={(e) => setKind(e.target.value as FormKind)}>
              {FORM_KINDS.map((k) => (
                <option key={k.value} value={k.value}>{k.label}</option>
              ))}
            </select>
            <span className="muted small">
              {kind === 'hardware_spec'
                ? 'این فرم در پرونده تجهیز به‌عنوان مشخصات سخت‌افزاری استفاده می‌شود'
                : 'این فرم برای ثبت گزارش فعالیت استفاده می‌شود'}
            </span>
          </div>
          <div className="form-row">
            <label>توضیح</label>
            <input className="input" value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>
        </div>
        {kind === 'report' && (
          <div className="form-row">
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <input
                type="checkbox"
                checked={requiresApproval}
                onChange={(e) => setRequiresApproval(e.target.checked)}
              />
              گزارش‌های این فرم نیاز به تایید مدیر دارند
            </label>
            <span className="muted small">
              اگر تیک را بردارید، گزارش‌های ثبت‌شده با این فرم در لحظه ثبت نهایی می‌شوند و مرحله تایید ندارند
            </span>
          </div>
        )}
        <div className="form-row">
          <label>فیلدهای فرم</label>
          <FieldsEditor fields={fields} onChange={setFields} />
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'در حال ایجاد...' : 'ایجاد فرم'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
