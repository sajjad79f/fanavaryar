import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { Asset, FormTemplate, FormVersion, Paged } from '../api/types'
import { PageHeader, Alert } from '../components/ui'
import { DynamicForm, validateDynamic } from '../components/DynamicForm'
import { JalaliInput } from '../components/JalaliInput'
import { useAuth } from '../context/AuthContext'
import { todayIso } from '../jalali'

export default function NewReport() {
  const { user } = useAuth()
  const navigate = useNavigate()

  const [templates, setTemplates] = useState<FormTemplate[]>([])
  const [templateId, setTemplateId] = useState('')
  const [version, setVersion] = useState<FormVersion | null>(null)
  const [title, setTitle] = useState('')
  const [dateIso, setDateIso] = useState(todayIso())
  const [formValues, setFormValues] = useState<Record<string, unknown>>({})
  const [selectedAssets, setSelectedAssets] = useState<Asset[]>([])
  const [assetQuery, setAssetQuery] = useState('')
  const [assetResults, setAssetResults] = useState<Asset[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    // فقط فرم‌های نوع «گزارش فعالیت»؛ فرم‌های مشخصات سخت‌افزاری در پرونده تجهیز استفاده می‌شوند
    api
      .get<FormTemplate[]>('/forms?kind=report')
      .then((list) => {
        const active = list.filter((t) => t.is_active)
        setTemplates(active)
        if (active.length > 0) setTemplateId(String(active[0].id))
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [])

  const loadVersion = useCallback(() => {
    const tpl = templates.find((t) => String(t.id) === templateId)
    if (!tpl?.current_version_id) {
      setVersion(null)
      return
    }
    api
      .get<FormVersion>(`/forms/versions/${tpl.current_version_id}`)
      .then((v) => {
        setVersion(v)
        setFormValues({})
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [templateId, templates])

  useEffect(() => {
    loadVersion()
  }, [loadVersion])

  const searchAssets = useCallback(() => {
    const params = new URLSearchParams({ per_page: '10' })
    if (assetQuery.trim()) params.set('q', assetQuery.trim())
    api
      .get<Paged<Asset>>(`/assets?${params}`)
      .then((p) => setAssetResults(p.items))
      .catch(() => setAssetResults([]))
  }, [assetQuery])

  useEffect(() => {
    searchAssets()
  }, [searchAssets])

  const selectedIds = useMemo(() => new Set(selectedAssets.map((a) => a.id)), [selectedAssets])

  function toggleAsset(a: Asset) {
    setSelectedAssets((prev) =>
      prev.some((x) => x.id === a.id) ? prev.filter((x) => x.id !== a.id) : [...prev, a],
    )
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!title.trim()) {
      setError('عنوان گزارش الزامی است')
      return
    }
    if (!version) {
      setError('ابتدا فرم را انتخاب کنید')
      return
    }
    if (!dateIso) {
      setError('تاریخ گزارش الزامی است')
      return
    }
    const clientErrors = validateDynamic(version.fields, formValues)
    if (clientErrors.length > 0) {
      setError(clientErrors.join('؛ '))
      return
    }
    setBusy(true)
    try {
      const res = await api.post<{ id: number }>('/reports', {
        title: title.trim(),
        form_template_version_id: version.id,
        form_data: formValues,
        asset_ids: selectedAssets.map((a) => a.id),
        report_date: dateIso,
      })
      navigate(`/reports/${res.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader title="ثبت گزارش جدید" subtitle="گزارش پس از ثبت در تاریخچه تجهیزات مرتبط نمایش داده می‌شود" />
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="two-cols">
          <div className="form-row">
            <label>عنوان گزارش *</label>
            <input className="input" value={title} onChange={(e) => setTitle(e.target.value)} />
          </div>
          <div className="form-row">
            <label>فرم *</label>
            <select className="input" value={templateId} onChange={(e) => setTemplateId(e.target.value)}>
              {templates.length === 0 && <option value="">فرم فعالی وجود ندارد</option>}
              {templates.map((t) => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="two-cols">
          <div className="form-row">
            <label>تاریخ گزارش (شمسی) *</label>
            <JalaliInput iso={dateIso} onIso={setDateIso} />
          </div>
          <div className="form-row">
            <label>ثبت‌کننده</label>
            <input className="input" value={user?.full_name ?? ''} disabled />
          </div>
        </div>

        {version && !version.requires_approval && (
          <Alert kind="success">
            این فرم نیاز به تایید مدیر ندارد؛ گزارش بلافاصله پس از ثبت نهایی می‌شود.
          </Alert>
        )}

        <section className="card">
          <div className="card-head"><h2>تجهیزات مرتبط (اختیاری)</h2></div>
          {selectedAssets.length > 0 && (
            <div className="chips">
              {selectedAssets.map((a) => (
                <span key={a.id} className="chip">
                  {a.asset_tag} — {a.name}
                  <button type="button" onClick={() => toggleAsset(a)}>×</button>
                </span>
              ))}
            </div>
          )}
          <div className="filter-bar">
            <input
              className="input grow"
              placeholder="جست‌وجوی تجهیز برای افزودن..."
              value={assetQuery}
              onChange={(e) => setAssetQuery(e.target.value)}
            />
          </div>
          <div className="asset-picker">
            {assetResults.map((a) => (
              <label key={a.id} className="asset-option">
                <input type="checkbox" checked={selectedIds.has(a.id)} onChange={() => toggleAsset(a)} />
                <span className="mono">{a.asset_tag}</span>
                <span>{a.name}</span>
                <span className="muted small">{a.location_name ?? ''}</span>
              </label>
            ))}
            {assetResults.length === 0 && <p className="muted">تجهیزی یافت نشد</p>}
          </div>
        </section>

        {version && (
          <section className="card">
            <div className="card-head">
              <h2>فیلدهای فرم: {version.fields.length} فیلد</h2>
            </div>
            <DynamicForm
              fields={version.fields}
              value={formValues}
              onChange={(key, value) => setFormValues((prev) => ({ ...prev, [key]: value }))}
            />
          </section>
        )}

        <div className="form-actions">
          <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'در حال ثبت...' : 'ثبت گزارش'}
          </button>
        </div>
      </form>
    </>
  )
}
