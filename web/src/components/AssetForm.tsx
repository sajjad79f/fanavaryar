import { useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import type { Asset, LocationRow } from '../api/types'
import { CATEGORIES, ASSET_STATUSES } from '../api/types'
import { Modal, Alert } from './ui'

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
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

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
    setBusy(true)
    const payload = {
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
