import { useCallback, useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import type { LocationRow, PermissionDef, PermissionTemplate, ScopeInfo, UserDetail } from '../api/types'
import { MODULE_LABELS, SCOPE_TYPE_LABELS } from '../api/types'
import { Modal, Alert } from './ui'

/**
 * ماتریس مجوزها — ماژول × عملیات، مطابق بخش ۳ سند دسترسی.
 * انتخاب الگو فقط ماتریس را پر می‌کند؛ ذخیره، مجوزهای مستقل کاربر را می‌نویسد
 * (الگو صرفاً ابزار تسریع است و جایگزین مجوزهای مستقل نمی‌شود).
 */
export function PermissionMatrix({
  catalog,
  selected,
  onToggle,
}: {
  catalog: PermissionDef[]
  selected: Set<string>
  onToggle: (code: string, on: boolean) => void
}) {
  const byModule = useMemo(() => {
    const map = new Map<string, PermissionDef[]>()
    for (const p of catalog) {
      const list = map.get(p.module) ?? []
      list.push(p)
      map.set(p.module, list)
    }
    return map
  }, [catalog])

  return (
    <div className="perms-matrix">
      {[...byModule.entries()].map(([module, perms]) => (
        <fieldset key={module} className="perms-module">
          <legend>{MODULE_LABELS[module] ?? module}</legend>
          {perms.map((p) => (
            <label key={p.code} className="check-label perm-item">
              <input
                type="checkbox"
                checked={selected.has(p.code)}
                onChange={(e) => onToggle(p.code, e.target.checked)}
              />
              <span>{p.title}</span>
            </label>
          ))}
        </fieldset>
      ))}
    </div>
  )
}

export default function PermsModal({
  userId,
  onClose,
  onSaved,
}: {
  userId: number
  onClose: () => void
  onSaved: () => void
}) {
  const [detail, setDetail] = useState<UserDetail | null>(null)
  const [catalog, setCatalog] = useState<PermissionDef[]>([])
  const [templates, setTemplates] = useState<PermissionTemplate[]>([])
  const [locations, setLocations] = useState<LocationRow[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [scopeType, setScopeType] = useState<ScopeInfo['scope_type']>('all')
  const [scopeLocationId, setScopeLocationId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    Promise.all([
      api.get<UserDetail>(`/users/${userId}`),
      api.get<PermissionDef[]>('/permissions'),
      api.get<PermissionTemplate[]>('/permission-templates'),
      api.get<LocationRow[]>('/locations'),
    ])
      .then(([d, cat, tpl, loc]) => {
        setDetail(d)
        setCatalog(cat)
        setTemplates(tpl)
        setLocations(loc)
        setSelected(new Set(d.permissions))
        if (d.scope) {
          setScopeType(d.scope.scope_type)
          setScopeLocationId(d.scope.location_id)
        }
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [userId])

  const toggle = useCallback((code: string, on: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (on) next.add(code)
      else next.delete(code)
      return next
    })
  }, [])

  function applyTemplate(name: string) {
    const tpl = templates.find((t) => t.name === name)
    if (tpl) setSelected(new Set(tpl.permissions))
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (scopeType !== 'all' && !scopeLocationId) {
      setError('برای محدودهٔ مکان‌محور، انتخاب مکان الزامی است')
      return
    }
    setBusy(true)
    try {
      await api.put(`/users/${userId}/permissions`, { permission_codes: [...selected] })
      await api.put(`/users/${userId}/scope`, {
        scope_type: scopeType,
        location_id: scopeType === 'all' ? null : scopeLocationId,
      })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  const scopeLocations = locations.filter((l) => l.kind === scopeType)
  const isAdmin = detail?.role === 'admin'

  return (
    <Modal title={`مجوزها و محدودهٔ دسترسی — ${detail?.full_name ?? '...'}`} onClose={onClose} width={780}>
      {error && <Alert kind="error">{error}</Alert>}
      {isAdmin && (
        <Alert kind="info">
          حساب مدیر سامانه از سیستم مجوز مستقل پیروی نمی‌کند و به همهٔ بخش‌ها دسترسی دارد.
        </Alert>
      )}
      {!isAdmin && (
        <form onSubmit={submit}>
          <div className="form-row">
            <label>اعمال الگوی دسترسی (اختیاری — فقط میان‌بر است)</label>
            <select
              className="input"
              value=""
              onChange={(e) => e.target.value && applyTemplate(e.target.value)}
            >
              <option value="">— انتخاب الگو —</option>
              {templates.map((t) => (
                <option key={t.id} value={t.name}>
                  {t.name} ({t.permissions.length} مجوز)
                </option>
              ))}
            </select>
          </div>

          <PermissionMatrix catalog={catalog} selected={selected} onToggle={toggle} />

          <fieldset className="perms-module">
            <legend>محدودهٔ دسترسی به تجهیزات</legend>
            <p className="muted small">
              کاربر مکانی‌محور فقط تجهیزاتِ ساختمان/طبقه/اتاق انتخابی و تجهیزاتِ بدون مکان را می‌بیند.
            </p>
            <div className="two-cols">
              <div className="form-row">
                <label>محدوده</label>
                <select
                  className="input"
                  value={scopeType}
                  onChange={(e) => {
                    setScopeType(e.target.value as ScopeInfo['scope_type'])
                    setScopeLocationId(null)
                  }}
                >
                  {SCOPE_TYPE_LABELS.map((s) => (
                    <option key={s.value} value={s.value}>{s.label}</option>
                  ))}
                </select>
              </div>
              {scopeType !== 'all' && (
                <div className="form-row">
                  <label>{SCOPE_TYPE_LABELS.find((s) => s.value === scopeType)?.label}</label>
                  <select
                    className="input"
                    value={scopeLocationId ?? ''}
                    onChange={(e) => setScopeLocationId(e.target.value ? Number(e.target.value) : null)}
                  >
                    <option value="">— انتخاب کنید —</option>
                    {scopeLocations.map((l) => (
                      <option key={l.id} value={l.id}>{l.name}</option>
                    ))}
                  </select>
                </div>
              )}
            </div>
          </fieldset>

          <div className="modal-actions">
            <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
            <button type="submit" className="btn btn-primary" disabled={busy}>
              {busy ? '...' : 'ذخیره دسترسی‌ها'}
            </button>
          </div>
        </form>
      )}
    </Modal>
  )
}
