import { useState } from 'react'
import type { FieldDef } from '../api/types'
import { Modal, Alert } from './ui'

const FIELD_TYPES: { value: FieldDef['type']; label: string }[] = [
  { value: 'text', label: 'متن کوتاه' },
  { value: 'textarea', label: 'متن بلند' },
  { value: 'number', label: 'عدد' },
  { value: 'date', label: 'تاریخ (شمسی)' },
  { value: 'select', label: 'انتخاب از گزینه‌ها' },
  { value: 'checkbox', label: 'بله/خیر' },
]

export const TYPE_LABELS: Record<string, string> = {
  text: 'متن کوتاه',
  textarea: 'متن بلند',
  number: 'عدد',
  date: 'تاریخ',
  select: 'انتخابی',
  checkbox: 'بله/خیر',
}

/** ویرایشگر ساختار فیلدهای فرم (JSON ساده با رابط جدولی) */
export function FieldsEditor({
  fields,
  onChange,
}: {
  fields: FieldDef[]
  onChange: (fields: FieldDef[]) => void
}) {
  const [editing, setEditing] = useState<{ index: number; draft: FieldDef } | null>(null)

  function addField() {
    setEditing({ index: -1, draft: { key: '', label: '', type: 'text', required: false, options: [] } })
  }

  function move(index: number, dir: -1 | 1) {
    const to = index + dir
    if (to < 0 || to >= fields.length) return
    const next = [...fields]
    ;[next[index], next[to]] = [next[to], next[index]]
    onChange(next)
  }

  function saveField(field: FieldDef) {
    if (editing?.index === -1) {
      onChange([...fields, field])
    } else if (editing) {
      const next = [...fields]
      next[editing.index] = field
      onChange(next)
    }
    setEditing(null)
  }

  return (
    <div className="fields-editor">
      <table className="table compact">
        <thead>
          <tr>
            <th>برچسب</th>
            <th>کلید</th>
            <th>نوع</th>
            <th>الزامی</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {fields.length === 0 && (
            <tr>
              <td colSpan={5}>
                <span className="muted">هنوز فیلدی اضافه نشده است</span>
              </td>
            </tr>
          )}
          {fields.map((f, i) => (
            <tr key={i}>
              <td>{f.label}</td>
              <td dir="ltr" className="mono">
                {f.key}
              </td>
              <td>{TYPE_LABELS[f.type] ?? f.type}</td>
              <td>{f.required ? 'بله' : 'خیر'}</td>
              <td className="row-actions">
                <button className="btn btn-ghost" onClick={() => setEditing({ index: i, draft: { ...f } })}>
                  ویرایش
                </button>
                <button className="btn btn-ghost" onClick={() => move(i, -1)} disabled={i === 0}>
                  ↑
                </button>
                <button className="btn btn-ghost" onClick={() => move(i, 1)} disabled={i === fields.length - 1}>
                  ↓
                </button>
                <button
                  className="btn btn-danger-ghost"
                  onClick={() => onChange(fields.filter((_, j) => j !== i))}
                >
                  حذف
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <button className="btn btn-secondary" onClick={addField}>
        افزودن فیلد
      </button>

      {editing && (
        <FieldEditModal draft={editing.draft} onClose={() => setEditing(null)} onSave={saveField} />
      )}
    </div>
  )
}

function FieldEditModal({
  draft,
  onClose,
  onSave,
}: {
  draft: FieldDef
  onClose: () => void
  onSave: (field: FieldDef) => void
}) {
  const [f, setF] = useState<FieldDef>({ ...draft, options: draft.options ?? [] })
  const [error, setError] = useState('')

  function save() {
    if (!f.label.trim()) {
      setError('برچسب فیلد الزامی است')
      return
    }
    if (!/^[a-z][a-z0-9_]*$/.test(f.key)) {
      setError('کلید باید با حرف انگلیسی کوچک شروع شود و فقط شامل حرف کوچک، رقم و زیرخط باشد')
      return
    }
    const options = (f.options ?? []).map((s) => s.trim()).filter(Boolean)
    if (f.type === 'select' && options.length === 0) {
      setError('برای فیلد انتخابی حداقل یک گزینه لازم است')
      return
    }
    if (f.type === 'number' && f.min !== undefined && f.max !== undefined && f.min > f.max) {
      setError('کمینه نباید از بیشینه بزرگ‌تر باشد')
      return
    }
    onSave({ ...f, options })
    onClose()
  }

  return (
    <Modal title="فیلد فرم" onClose={onClose}>
      {error && <Alert kind="error">{error}</Alert>}
      <div className="form-row">
        <label>برچسب (نام نمایشی)</label>
        <input className="input" value={f.label} onChange={(e) => setF({ ...f, label: e.target.value })} />
      </div>
      <div className="form-row">
        <label>کلید (انگلیسی، یکتا)</label>
        <input className="input" dir="ltr" value={f.key} onChange={(e) => setF({ ...f, key: e.target.value })} />
      </div>
      <div className="form-row">
        <label>نوع فیلد</label>
        <select
          className="input"
          value={f.type}
          onChange={(e) => setF({ ...f, type: e.target.value as FieldDef['type'] })}
        >
          {FIELD_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </div>
      {f.type === 'number' && (
        <div className="two-cols">
          <div className="form-row">
            <label>کمینه (اختیاری)</label>
            <input
              className="input"
              type="number"
              value={f.min ?? ''}
              onChange={(e) => setF({ ...f, min: e.target.value === '' ? undefined : Number(e.target.value) })}
            />
          </div>
          <div className="form-row">
            <label>بیشینه (اختیاری)</label>
            <input
              className="input"
              type="number"
              value={f.max ?? ''}
              onChange={(e) => setF({ ...f, max: e.target.value === '' ? undefined : Number(e.target.value) })}
            />
          </div>
        </div>
      )}
      {f.type === 'select' && (
        <div className="form-row">
          <label>گزینه‌ها (هر خط یک گزینه)</label>
          <textarea
            className="input"
            rows={4}
            value={(f.options ?? []).join('\n')}
            onChange={(e) => setF({ ...f, options: e.target.value.split('\n') })}
          />
        </div>
      )}
      <div className="form-row">
        <label className="check-label">
          <input type="checkbox" checked={!!f.required} onChange={(e) => setF({ ...f, required: e.target.checked })} />
          <span>الزامی است</span>
        </label>
      </div>
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onClose}>
          انصراف
        </button>
        <button className="btn btn-primary" onClick={save}>
          ذخیره
        </button>
      </div>
    </Modal>
  )
}
