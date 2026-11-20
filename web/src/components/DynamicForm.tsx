import { formatJalali } from '../jalali'
import type { FieldDef } from '../api/types'
import { parseJalali, isoToJalaliText } from '../jalali'
import { JalaliInput } from './JalaliInput'

interface Props {
  fields: FieldDef[]
  value: Record<string, unknown>
  onChange: (key: string, value: unknown) => void
  disabled?: boolean
}

export function DynamicForm({ fields, value, onChange, disabled }: Props) {
  return (
    <div className="dynamic-form">
      {fields.map((f) => (
        <div className="form-row" key={f.key}>
          <label>
            {f.label}
            {f.required && <span className="req"> *</span>}
          </label>
          {f.type === 'textarea' && (
            <textarea
              className="input"
              rows={3}
              disabled={disabled}
              value={String(value[f.key] ?? '')}
              onChange={(e) => onChange(f.key, e.target.value)}
            />
          )}
          {f.type === 'text' && (
            <input
              className="input"
              type="text"
              disabled={disabled}
              value={String(value[f.key] ?? '')}
              onChange={(e) => onChange(f.key, e.target.value)}
            />
          )}
          {f.type === 'number' && (
            <input
              className="input"
              type="number"
              disabled={disabled}
              min={f.min}
              max={f.max}
              value={value[f.key] === undefined || value[f.key] === null ? '' : String(value[f.key])}
              onChange={(e) =>
                onChange(f.key, e.target.value === '' ? undefined : Number(e.target.value))
              }
            />
          )}
          {f.type === 'date' && (
            <JalaliInput
              iso={typeof value[f.key] === 'string' ? (value[f.key] as string) : ''}
              onIso={(iso) => onChange(f.key, iso || undefined)}
              disabled={disabled}
            />
          )}
          {f.type === 'select' && (
            <select
              className="input"
              disabled={disabled}
              value={String(value[f.key] ?? '')}
              onChange={(e) => onChange(f.key, e.target.value === '' ? undefined : e.target.value)}
            >
              <option value="">— انتخاب کنید —</option>
              {(f.options ?? []).map((o) => (
                <option key={o} value={o}>
                  {o}
                </option>
              ))}
            </select>
          )}
          {f.type === 'checkbox' && (
            <label className="check-label">
              <input
                type="checkbox"
                disabled={disabled}
                checked={value[f.key] === true}
                onChange={(e) => onChange(f.key, e.target.checked)}
              />
              <span>{f.label}</span>
            </label>
          )}
        </div>
      ))}
    </div>
  )
}

/** اعتبارسنجی سمت کلاینت — هم‌راستا با قواعد بک‌اند */
export function validateDynamic(fields: FieldDef[], value: Record<string, unknown>): string[] {
  const errors: string[] = []
  for (const f of fields) {
    const v = value[f.key]
    const empty = v === undefined || v === null || v === ''
    if (f.required && empty) {
      errors.push(`فیلد «${f.label}» الزامی است`)
      continue
    }
    if (!empty) {
      if (f.type === 'number') {
        const n = Number(v)
        if (isNaN(n)) errors.push(`فیلد «${f.label}» باید عدد باشد`)
        else if (f.min !== undefined && n < f.min) errors.push(`فیلد «${f.label}» باید بیشتر یا مساوی ${f.min} باشد`)
        else if (f.max !== undefined && n > f.max) errors.push(`فیلد «${f.label}» باید کمتر یا مساوی ${f.max} باشد`)
      }
      if (f.type === 'date' && typeof v === 'string' && !parseJalali(isoToJalaliText(v))) {
        errors.push(`تاریخ فیلد «${f.label}» نامعتبر است`)
      }
      if (f.type === 'select' && typeof v === 'string' && !(f.options ?? []).includes(v)) {
        errors.push(`مقدار فیلد «${f.label}» باید از گزینه‌های تعریف‌شده باشد`)
      }
    }
  }
  return errors
}

// برای نمایش مقدار ثبت‌شده در نمای گزارش
export function displayValue(f: FieldDef, v: unknown): string {
  if (v === undefined || v === null || v === '') return '—'
  if (f.type === 'checkbox') return v === true ? 'بله' : 'خیر'
  if (f.type === 'date') return formatJalali(String(v))
  return String(v)
}
