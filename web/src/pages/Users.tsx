import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { UserItem, Paged } from '../api/types'
import { PageHeader, Modal, Alert, Pagination } from '../components/ui'
import { formatJalaliDateTime } from '../jalali'
import { can, hasPerm, useAuth } from '../context/AuthContext'
import PermsModal from '../components/PermsModal'
import TemplatesModal from '../components/TemplatesModal'

export default function Users() {
  const { user: me } = useAuth()
  const [q, setQ] = useState('')
  const [page, setPage] = useState(1)
  const [data, setData] = useState<Paged<UserItem> | null>(null)
  const [error, setError] = useState('')
  const [modal, setModal] = useState<
    | { kind: 'create' }
    | { kind: 'edit'; user: UserItem }
    | { kind: 'reset'; user: UserItem }
    | { kind: 'perms'; user: UserItem }
    | { kind: 'templates' }
    | null
  >(null)

  const canCreate = hasPerm(me, 'users.create')
  const canEdit = hasPerm(me, 'users.edit')
  const canDeactivate = hasPerm(me, 'users.deactivate')
  const canPerms = can(me, 'manage_user_perms')

  const load = useCallback(() => {
    const params = new URLSearchParams({ page: String(page), per_page: '20' })
    if (q.trim()) params.set('q', q.trim())
    api
      .get<Paged<UserItem>>(`/users?${params}`)
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [q, page])

  useEffect(() => {
    load()
  }, [load])

  return (
    <>
      <PageHeader
        title="کاربران"
        subtitle="مدیریت کاربران، مجوزهای ریزدانه و محدودهٔ دسترسی"
        actions={
          <>
            {canPerms && (
              <button className="btn btn-secondary" onClick={() => setModal({ kind: 'templates' })}>
                الگوهای دسترسی
              </button>
            )}
            {canCreate && (
              <button className="btn btn-primary" onClick={() => setModal({ kind: 'create' })}>
                کاربر جدید
              </button>
            )}
          </>
        }
      />
      <div className="filter-bar">
        <input
          className="input grow"
          placeholder="جست‌وجو: نام کاربری یا نام و نام خانوادگی..."
          value={q}
          onChange={(e) => {
            setQ(e.target.value)
            setPage(1)
          }}
        />
      </div>
      {error && <Alert kind="error">{error}</Alert>}
      {data && (
        <>
          <table className="table card">
            <thead>
              <tr>
                <th>نام کاربری</th>
                <th>نام و نام خانوادگی</th>
                <th>سمت سازمانی</th>
                <th>نوع حساب</th>
                <th>وضعیت</th>
                <th>تاریخ ایجاد</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((u) => {
                const isAdmin = u.role === 'admin'
                const canTouch = !isAdmin || me?.is_superadmin
                return (
                  <tr key={u.id} className={!u.is_active ? 'dim' : ''}>
                    <td dir="ltr" className="mono">{u.username}</td>
                    <td>{u.full_name}</td>
                    <td>{u.job_title ?? '—'}</td>
                    <td>
                      {isAdmin
                        ? <span className="badge badge-success">مدیر سامانه</span>
                        : <span className="badge badge-muted">کاربر مجوزمحور</span>}
                    </td>
                    <td>
                      {u.is_active ? <span className="badge badge-success">فعال</span> : <span className="badge badge-muted">غیرفعال</span>}
                    </td>
                    <td>{formatJalaliDateTime(u.created_at)}</td>
                    <td className="row-actions">
                      {canPerms && canTouch && !isAdmin && (
                        <button className="btn btn-ghost small" onClick={() => setModal({ kind: 'perms', user: u })}>
                          دسترسی‌ها
                        </button>
                      )}
                      {canEdit && canTouch && (
                        <button className="btn btn-ghost small" onClick={() => setModal({ kind: 'edit', user: u })}>
                          ویرایش
                        </button>
                      )}
                      {canEdit && canTouch && (
                        <button className="btn btn-ghost small" onClick={() => setModal({ kind: 'reset', user: u })}>
                          بازنشانی گذرواژه
                        </button>
                      )}
                      {!canPerms && !canEdit && <span className="muted small">—</span>}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          <Pagination page={data.page} perPage={data.per_page} total={data.total} onChange={setPage} />
        </>
      )}

      {modal?.kind === 'create' && (
        <UserFormModal
          onClose={() => setModal(null)}
          onSaved={() => {
            setModal(null)
            load()
          }}
        />
      )}
      {modal?.kind === 'edit' && (
        <EditUserModal
          target={modal.user}
          meId={me?.id ?? 0}
          canDeactivate={canDeactivate}
          onClose={() => setModal(null)}
          onSaved={() => {
            setModal(null)
            load()
          }}
        />
      )}
      {modal?.kind === 'reset' && (
        <ResetPasswordModal
          target={modal.user}
          onClose={() => setModal(null)}
          onSaved={() => {
            setModal(null)
            load()
          }}
        />
      )}
      {modal?.kind === 'perms' && (
        <PermsModal
          userId={modal.user.id}
          onClose={() => setModal(null)}
          onSaved={() => {
            setModal(null)
            load()
          }}
        />
      )}
      {modal?.kind === 'templates' && (
        <TemplatesModal onClose={() => setModal(null)} />
      )}
    </>
  )
}

function UserFormModal({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const [username, setUsername] = useState('')
  const [fullName, setFullName] = useState('')
  const [jobTitle, setJobTitle] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await api.post('/users', {
        username: username.trim(),
        full_name: fullName.trim(),
        job_title: jobTitle.trim(),
        password,
      })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="کاربر جدید" onClose={onClose}>
      <Alert kind="info">
        پس از ایجاد کاربر، از دکمهٔ «دسترسی‌ها» مجوزهای او را متناسب با مسئولیتش تعیین کنید
        (اصل حداقل دسترسی). سمت سازمانی صرفاً اطلاعاتی است و دسترسی نمی‌سازد.
      </Alert>
      <form onSubmit={submit}>
        <div className="two-cols">
          <div className="form-row">
            <label>نام کاربری (انگلیسی) *</label>
            <input className="input" dir="ltr" value={username} onChange={(e) => setUsername(e.target.value)} />
          </div>
          <div className="form-row">
            <label>نام و نام خانوادگی *</label>
            <input className="input" value={fullName} onChange={(e) => setFullName(e.target.value)} />
          </div>
        </div>
        <div className="two-cols">
          <div className="form-row">
            <label>سمت سازمانی</label>
            <input className="input" placeholder="مثلاً کارشناس شبکه" value={jobTitle} onChange={(e) => setJobTitle(e.target.value)} />
          </div>
          <div className="form-row">
            <label>گذرواژه (حداقل ۸ نویسه) *</label>
            <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>{busy ? '...' : 'ذخیره'}</button>
        </div>
      </form>
    </Modal>
  )
}

function EditUserModal({
  target,
  meId,
  canDeactivate,
  onClose,
  onSaved,
}: {
  target: UserItem
  meId: number
  canDeactivate: boolean
  onClose: () => void
  onSaved: () => void
}) {
  const [fullName, setFullName] = useState(target.full_name)
  const [jobTitle, setJobTitle] = useState(target.job_title ?? '')
  const [isActive, setIsActive] = useState(target.is_active)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const isSelf = target.id === meId
  const activeChanged = isActive !== target.is_active

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const body: Record<string, unknown> = {
        full_name: fullName.trim(),
        job_title: jobTitle.trim(),
      }
      if (activeChanged) body.is_active = isActive
      await api.patch(`/users/${target.id}`, body)
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={`ویرایش کاربر ${target.username}`} onClose={onClose}>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="form-row">
          <label>نام و نام خانوادگی</label>
          <input className="input" value={fullName} onChange={(e) => setFullName(e.target.value)} />
        </div>
        <div className="form-row">
          <label>سمت سازمانی</label>
          <input className="input" value={jobTitle} onChange={(e) => setJobTitle(e.target.value)} />
        </div>
        {canDeactivate && (
          <div className="form-row">
            <label className="check-label">
              <input
                type="checkbox"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
                disabled={isSelf}
              />
              <span>فعال</span>
            </label>
            {isSelf && <span className="muted small">تغییر وضعیت حساب خودتان مجاز نیست</span>}
          </div>
        )}
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>{busy ? '...' : 'ذخیره'}</button>
        </div>
      </form>
    </Modal>
  )
}

function ResetPasswordModal({
  target,
  onClose,
  onSaved,
}: {
  target: UserItem
  onClose: () => void
  onSaved: () => void
}) {
  const [newPassword, setNewPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (newPassword.length < 8) {
      setError('گذرواژه باید حداقل ۸ نویسه باشد')
      return
    }
    setBusy(true)
    try {
      await api.post(`/users/${target.id}/reset-password`, { new_password: newPassword })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={`بازنشانی گذرواژه ${target.username}`} onClose={onClose}>
      <p className="muted small">پس از بازنشانی، همه نشست‌های فعال این کاربر باطل می‌شود.</p>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="form-row">
          <label>گذرواژه جدید (حداقل ۸ نویسه) *</label>
          <input className="input" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>{busy ? '...' : 'بازنشانی'}</button>
        </div>
      </form>
    </Modal>
  )
}
