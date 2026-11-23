import { useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import type { Asset, AssetDetail, FieldDef, FormTemplate, FormVersion, LocationRow } from '../api/types'
import { CATEGORIES, ASSET_STATUSES } from '../api/types'
import { Modal, Alert } from './ui'
import { DynamicForm, validateDynamic } from './DynamicForm'

export interface LocationOption {
  id: number
  name: string
  depth: number
}

/** تبدیل لیست تخت مکان‌ها به گزینه‌های مرتب درختی برای select */
export function buildLocationOptions(locations: LocationRow[]): LocationOption[] {
  const byId = new Map(locations.map((l) => [l.id, l]))
  const depthOf = (l: LocationRow): number => {
    let d = 0
    let cur = l.parent_id
    while (cur !== null) {
      d += 1
      cur = byId.get(cur)?.parent_id ?? null
    }
    return d
  }
  const sorted = [...locations].sort((a, b) => {
    const da = depthOf(a)
    const db = depthOf(b)
    if (da !== db) return da - db
    return a.name.localeCompare(b.name, 'fa')
  })
  return sorted.map((l) => ({ id: l.id, name: l.name, depth: depthOf(l) }))
}

export interface AssetFormValues {
  asset_tag: string
  name: string
  category: string
  status: string
  serial_number: string
  manufacturer: string
  model: string
  responsible_person: string
  current_location_id: string
  notes: string
}

const emptyValues: AssetFormValues = {
  asset_tag: '',
  name: '',
  category: 'desktop',
  status: 'active',
  serial_number: '',
  manufacturer: '',
  model: '',
  responsible_person: '',
  current_location_id: '',
  notes: '',
}

/** مودال ساخت/ویرایش تجهیز — در لیست تجهیزات و صفحه پرونده تجهیز استفاده می‌شود */
export function AssetFormModal({
  asset,
  locations,
  onClose,
  onSaved,
}: {
  asset?: Asset
  locations: LocationOption[]
  onClose: () => void
  onSaved: () => void
}) {
  const detail = asset as AssetDetail | undefined
  const hadSpecs = !!asset?.spec_form_version_id
  // فیلدها و داده مشخصات فقط در پاسخ Detail هست؛ از صفحه پرونده کامل است، از لیست نه
  const specPreloaded = hadSpecs && Array.isArray(detail?.spec_fields)

  const [values, setValues] = useState<AssetFormValues>(
    asset
      ? {
          asset_tag: asset.asset_tag,
          name: asset.name,
          category: asset.category,
          status: asset.status,
          serial_number: asset.serial_number ?? '',
          manufacturer: asset.manufacturer ?? '',
          model: asset.model ?? '',
          responsible_person: asset.responsible_person ?? '',
          current_location_id: asset.current_location_id ? String(asset.current_location_id) : '',
          notes: asset.notes ?? '',
        }
      : emptyValues,
  )
  // ---------- مشخصات فرمی سخت‌افزاری (اختیاری) ----------
  const [specForms, setSpecForms] = useState<FormTemplate[]>([])
  const [specVersionId, setSpecVersionId] = useState(hadSpecs ? String(asset.spec_form_version_id) : '')
  const [specFields, setSpecFields] = useState<FieldDef[]>([])
  const [specValues, setSpecValues] = useState<Record<string, unknown>>({})
  const [specTouched, setSpecTouched] = useState(false)
  const [clearSpecs, setClearSpecs] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api
      .get<FormTemplate[]>('/forms?kind=hardware_spec')
      .then((list) => setSpecForms(list.filter((t) => t.is_active && t.current_version_id)))
      .catch(() => {})
  }, [])

  useEffect(() => {
    // پیش‌بارگذاری مشخصات موجود (فقط از صفحه پرونده که فیلدها و داده را دارد)
    if (specPreloaded && detail) {
      setSpecFields(detail.spec_fields)
      setSpecValues(detail.spec_data ?? {})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function onSpecFormChange(v: string) {
    if (v === specVersionId) return
    setSpecTouched(true)
    setClearSpecs(hadSpecs && v === '')
    setSpecVersionId(v)
    if (!v) {
      setSpecFields([])
      setSpecValues({})
      return
    }
    try {
      const ver = await api.get<FormVersion>(`/forms/versions/${v}`)
      setSpecFields(ver.fields)
      setSpecValues({})
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    }
  }

  function set<K extends keyof AssetFormValues>(key: K, v: string) {
    setValues((prev) => ({ ...prev, [key]: v }))
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!values.asset_tag.trim() || !values.name.trim()) {
      setError('شناسه و نام تجهیز الزامی است')
      return
    }
    const payload: Record<string, unknown> = {
      asset_tag: values.asset_tag.trim(),
      name: values.name.trim(),
      category: values.category,
      status: values.status,
      serial_number: values.serial_number.trim() || undefined,
      manufacturer: values.manufacturer.trim() || undefined,
      model: values.model.trim() || undefined,
      responsible_person: values.responsible_person.trim() || undefined,
      current_location_id: values.current_location_id ? Number(values.current_location_id) : undefined,
      notes: values.notes.trim() || undefined,
    }
    // مشخصات فرمی: فقط وقتی کاربر تغییرش داده ارسال می‌شود تا داده موجود بی‌جهت بازنویسی نشود
    if (clearSpecs) {
      payload.clear_specs = true
    } else if (specVersionId && specTouched) {
      const specErrors = validateDynamic(specFields, specValues)
      if (specErrors.length > 0) {
        setError(specErrors.join('؛ '))
        return
      }
      payload.spec_form_version_id = Number(specVersionId)
      payload.spec_data = specValues
    }
    setBusy(true)
    try {
      if (asset) {
        await api.patch(`/assets/${asset.id}`, payload)
      } else {
        await api.post('/assets', payload)
      }
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={asset ? 'ویرایش تجهیز' : 'ثبت تجهیز جدید'} onClose={onClose} width={620}>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="two-cols">
          <div className="form-row">
            <label>شناسه / کد اموال *</label>
            <input className="input" dir="ltr" value={values.asset_tag} onChange={(e) => set('asset_tag', e.target.value)} />
          </div>
          <div className="form-row">
            <label>نام تجهیز *</label>
            <input className="input" value={values.name} onChange={(e) => set('name', e.target.value)} />
          </div>
        </div>
        <div className="two-cols">
          <div className="form-row">
            <label>دسته‌بندی</label>
            <select className="input" value={values.category} onChange={(e) => set('category', e.target.value)}>
              {CATEGORIES.map((c) => (
                <option key={c.value} value={c.value}>{c.label}</option>
              ))}
            </select>
          </div>
          <div className="form-row">
            <label>وضعیت</label>
            <select className="input" value={values.status} onChange={(e) => set('status', e.target.value)}>
              {ASSET_STATUSES.map((s) => (
                <option key={s.value} value={s.value}>{s.label}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="two-cols">
          <div className="form-row">
            <label>شماره سریال</label>
            <input className="input" dir="ltr" value={values.serial_number} onChange={(e) => set('serial_number', e.target.value)} />
          </div>
          <div className="form-row">
            <label>سازنده</label>
            <input className="input" value={values.manufacturer} onChange={(e) => set('manufacturer', e.target.value)} />
          </div>
        </div>
        <div className="two-cols">
          <div className="form-row">
            <label>مدل</label>
            <input className="input" value={values.model} onChange={(e) => set('model', e.target.value)} />
          </div>
          <div className="form-row">
            <label>مسئول / تحویل‌گیرنده</label>
            <input className="input" value={values.responsible_person} onChange={(e) => set('responsible_person', e.target.value)} />
          </div>
        </div>
        <div className="form-row">
          <label>مکان فعلی</label>
          <select className="input" value={values.current_location_id} onChange={(e) => set('current_location_id', e.target.value)} disabled={!!asset}>
            <option value="">— بدون مکان —</option>
            {locations.map((o) => (
              <option key={o.id} value={o.id}>{'— '.repeat(o.depth)}{o.name}</option>
            ))}
          </select>
          {asset && <span className="hint">تغییر مکان از طریق دکمه «جابه‌جایی» در پرونده تجهیز انجام می‌شود تا در تاریخچه ثبت شود.</span>}
        </div>
        <div className="form-row">
          <label>توضیحات</label>
          <textarea className="input" rows={2} value={values.notes} onChange={(e) => set('notes', e.target.value)} />
        </div>
        <div className="form-row">
          <label>فرم مشخصات سخت‌افزاری</label>
          {hadSpecs && !specPreloaded ? (
            <>
              <input className="input" value="مشخصات فرمی ثبت شده است" disabled />
              <span className="muted small">ویرایش مشخصات فرمی از صفحه پرونده تجهیز انجام می‌شود</span>
            </>
          ) : (
            <>
              <select
                className="input"
                value={specVersionId}
                onChange={(e) => onSpecFormChange(e.target.value)}
                disabled={clearSpecs}
              >
                <option value="">— بدون فرم مشخصات —</option>
                {specForms.map((t) => (
                  <option key={t.id} value={String(t.current_version_id)}>{t.name}</option>
                ))}
                {/* تجهیز ممکن است به نسخه قدیمی‌تری از فرم قفل شده باشد */}
                {specPreloaded &&
                  specVersionId !== '' &&
                  !specForms.some((t) => String(t.current_version_id) === specVersionId) && (
                    <option value={specVersionId}>
                      {detail?.spec_template_name} (نسخه ثبت‌شده {detail?.spec_version_number})
                    </option>
                  )}
              </select>
              <span className="muted small">
                اختیاری — فیلدهای فرم انتخابی در پرونده تجهیز نمایش داده می‌شوند
                {hadSpecs && '؛ انتخاب «بدون فرم مشخصات» یعنی حذف مشخصات فعلی'}
              </span>
              {clearSpecs && <Alert kind="info">با ذخیره، مشخصات فرمی این تجهیز حذف می‌شود</Alert>}
              {specFields.length > 0 && (
                <DynamicForm
                  fields={specFields}
                  value={specValues}
                  onChange={(key, value) => {
                    setSpecValues((prev) => ({ ...prev, [key]: value }))
                    setSpecTouched(true)
                  }}
                />
              )}
            </>
          )}
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'در حال ذخیره...' : 'ذخیره'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

/** هوک دریافت مکان‌ها به‌صورت گزینه‌های درختی */
export function useLocationOptions(): LocationOption[] {
  const [locations, setLocations] = useState<LocationRow[]>([])
  useEffect(() => {
    api.get<LocationRow[]>('/locations').then(setLocations).catch(() => {})
  }, [])
  return useMemo(() => buildLocationOptions(locations), [locations])
}
