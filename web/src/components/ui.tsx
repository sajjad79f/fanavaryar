import type { ReactNode } from 'react'

export function PageHeader({
  title,
  actions,
  subtitle,
}: {
  title: string
  actions?: ReactNode
  subtitle?: string
}) {
  return (
    <div className="page-header">
      <div>
        <h1>{title}</h1>
        {subtitle && <p className="page-subtitle">{subtitle}</p>}
      </div>
      <div className="page-actions">{actions}</div>
    </div>
  )
}

export function Modal({
  title,
  onClose,
  children,
  width = 520,
}: {
  title: string
  onClose: () => void
  children: ReactNode
  width?: number
}) {
  return (
    <div className="modal-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal" style={{ maxWidth: width }}>
        <div className="modal-head">
          <h3>{title}</h3>
          <button className="btn-icon" onClick={onClose} aria-label="بستن">
            ✕
          </button>
        </div>
        <div className="modal-body">{children}</div>
      </div>
    </div>
  )
}

export function Alert({ kind, children }: { kind: 'error' | 'success' | 'info'; children: ReactNode }) {
  return <div className={`alert alert-${kind}`}>{children}</div>
}

export function Spinner() {
  return <span className="spinner" aria-label="در حال بارگذاری" />
}

export function EmptyState({ text }: { text: string }) {
  return <div className="empty-state">{text}</div>
}

export function Pagination({
  page,
  perPage,
  total,
  onChange,
}: {
  page: number
  perPage: number
  total: number
  onChange: (p: number) => void
}) {
  const pages = Math.max(1, Math.ceil(total / perPage))
  if (pages <= 1) return null
  return (
    <div className="pagination">
      <button className="btn btn-ghost" disabled={page <= 1} onClick={() => onChange(page - 1)}>
        قبلی
      </button>
      <span>
        صفحه {page} از {pages}
      </span>
      <button className="btn btn-ghost" disabled={page >= pages} onClick={() => onChange(page + 1)}>
        بعدی
      </button>
    </div>
  )
}

export function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; cls: string }> = {
    pending: { label: 'در انتظار تایید', cls: 'badge-warning' },
    approved: { label: 'تایید شده', cls: 'badge-success' },
    rejected: { label: 'رد شده', cls: 'badge-danger' },
    active: { label: 'فعال', cls: 'badge-success' },
    repair: { label: 'در تعمیر', cls: 'badge-warning' },
    storage: { label: 'انبار', cls: 'badge-muted' },
    retired: { label: 'از رده خارج', cls: 'badge-danger' },
  }
  const v = map[status]
  if (!v) return <span className="badge badge-muted">{status}</span>
  return <span className={`badge ${v.cls}`}>{v.label}</span>
}
