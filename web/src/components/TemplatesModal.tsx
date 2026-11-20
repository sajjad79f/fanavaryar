import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { PermissionDef, PermissionTemplate } from '../api/types'
import { Modal, Alert } from './ui'
import { PermissionMatrix } from './PermsModal'

/** مدیریت الگوهای دسترسی — بخش ۵ سند؛ فقط ابزار تسریع تخصیص مجوزها */
export default function TemplatesModal({ onClose }: { onClose: () => void }) {
  const [templates, setTemplates] = useState<PermissionTemplate[]>([])
  const [catalog, setCatalog] = useState<PermissionDef[]>([])
  const [name, setName] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => {
    Promise.all([
      api.get<PermissionTemplate[]>('/permission-templates'),
      api.get<PermissionDef[]>('/permissions'),
    ])
      .then(([tpl, cat]) => {
        setTemplates(tpl)
        setCatalog(cat)
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  function toggle(code: string, on: boolean) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (on) next.add(code)
      else next.delete(code)
      return next
    })
  }

  async function createTemplate(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setNotice('')
    setBusy(true)
    try {
      await api.post('/permission-templates', {
        name: name.trim(),
        permission_codes: [...selected],
      })
      setName('')
      setSelected(new Set())
      setNotice('الگو ذخیره شد')
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  async function removeTemplate(id: number, tplName: string) {
    if (!window.confirm(`حذف الگوی «${tplName}»؟ مجوزهای کاربران تغییر نمی‌کند.`)) return
    setError('')
    try {
      await api.delete(`/permission-templates/${id}`)
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    }
  }

  return (
    <Modal title="الگوهای دسترسی" onClose={onClose} width={780}>
      {error && <Alert kind="error">{error}</Alert>}
      {notice && <Alert kind="success">{notice}</Alert>}

      <table className="table compact card">
        <thead>
          <tr>
            <th>نام الگو</th>
            <th>تعداد مجوز</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {templates.map((t) => (
            <tr key={t.id}>
              <td>{t.name}</td>
              <td>{t.permissions.length}</td>
              <td className="row-actions">
                <button className="btn btn-danger-ghost small" onClick={() => removeTemplate(t.id, t.name)}>
                  حذف
                </button>
              </td>
            </tr>
          ))}
          {templates.length === 0 && (
            <tr><td colSpan={3}><span className="muted">الگویی ثبت نشده است</span></td></tr>
          )}
        </tbody>
      </table>

      <form onSubmit={createTemplate}>
        <div className="form-row">
          <label>نام الگوی جدید * (مانند: کارشناس شبکه)</label>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <PermissionMatrix catalog={catalog} selected={selected} onToggle={toggle} />
        <div className="modal-actions">
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? '...' : 'ذخیره الگو'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
