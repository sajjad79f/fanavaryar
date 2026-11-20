import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { FormTemplate } from '../api/types'
import { PageHeader, Modal, Alert } from '../components/ui'
import { FieldsEditor } from '../components/FieldsEditor'
import type { FieldDef } from '../api/types'
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
              <td>{t.description ?? '—'}</td>
              <td>
                {t.is_active ? <span className="badge badge-success">فعال</span> : <span className="badge badge-muted">غیرفعال</span>}
              </td>
              <td>{formatJalaliDateTime(t.created_at)}</td>
            </tr>
          ))}
          {templates.length === 0 && (
            <tr>
              <td colSpan={5}><span className="muted">هنوز فرمی تعریف نشده است</span></td>
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
            <input className="input" dir="ltr" placeholder="it-service-form" value={code} onChange={(e) => setCode(e.target.value)} />
          </div>
          <div className="form-row">
            <label>نام فرم *</label>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
        </div>
        <div className="form-row">
          <label>توضیح</label>
          <input className="input" value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
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
