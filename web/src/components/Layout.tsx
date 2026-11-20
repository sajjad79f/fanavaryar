import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth, can } from '../context/AuthContext'
import { api } from '../api/client'
import { Modal, Alert } from './ui'

const NAV = [
  { to: '/', label: 'داشبورد', end: true, perm: null },
  { to: '/assets', label: 'تجهیزات', end: false, perm: null },
  { to: '/locations', label: 'مکان‌ها', end: false, perm: null },
  { to: '/forms', label: 'فرم‌ها', end: false, perm: null },
  { to: '/reports', label: 'گزارش‌ها', end: false, perm: null },
  { to: '/users', label: 'کاربران', end: false, perm: 'manage_users' },
  { to: '/audit', label: 'دفتر رویدادها', end: false, perm: 'view_audit' },
]

const SIMPLE_ICONS: Record<string, string> = {
  '/': 'M3 12h6V3H3v9zm0 9h6v-6H3v6zm9 0h9V10h-9v11zm0-18v5h9V3h-9z',
  '/assets': 'M4 4h6v6H4zM10 10h6v6h-6zM16 4h4v4h-4zM4 14h4v6H4z',
  '/locations': 'M12 2 4 8h2v12h4v-6h4v6h4V8h2L12 2z',
  '/forms': 'M5 3h10l4 4v14H5V3zm3 6h8v2H8V9zm0 4h8v2H8v-2zm0 4h6v2H8v-2z',
  '/reports': 'M6 2h9l5 5v15H6V2zm8 1v5h5M8 12h8v2H8zm0 4h8v2H8z',
  '/users': 'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zm0 2c-4 0-7 2-7 5v3h14v-3c0-3-3-5-7-5z',
  '/audit': 'M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20zm1 5h-2v6l5 3 1-1.7-4-2.3V7z',
}

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [showPasswordModal, setShowPasswordModal] = useState(false)

  async function handleLogout() {
    await logout()
    navigate('/login')
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">ف</div>
          <div>
            <div className="brand-name">فناوریار</div>
            <div className="brand-sub">سامانه ثبت سوابق IT</div>
          </div>
        </div>
        <nav>
          {NAV.filter((item) => !item.perm || can(user, item.perm)).map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end} className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}>
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" aria-hidden>
                <path d={SIMPLE_ICONS[item.to]} />
              </svg>
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-foot">نسخه ۰٫۱ — فاز ۱</div>
      </aside>

      <div className="main">
        <header className="topbar">
          <div className="topbar-spacer" />
          <div className="topbar-user">
            <span className="user-name">
              {user?.full_name}
              <span className="user-role">
                {user?.is_superadmin || user?.role === 'admin'
                  ? 'مدیر سامانه'
                  : (user?.job_title ?? 'کاربر مجوزمحور')}
              </span>
            </span>
            <button className="btn btn-ghost" onClick={() => setShowPasswordModal(true)}>
              تغییر گذرواژه
            </button>
            <button className="btn btn-ghost" onClick={handleLogout}>
              خروج
            </button>
          </div>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>

      {showPasswordModal && <ChangePasswordModal onClose={() => setShowPasswordModal(false)} />}
    </div>
  )
}

function ChangePasswordModal({ onClose }: { onClose: () => void }) {
  const [oldPassword, setOld] = useState('')
  const [newPassword, setNew] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [ok, setOk] = useState(false)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (newPassword.length < 8) {
      setError('گذرواژه جدید باید حداقل ۸ نویسه باشد')
      return
    }
    if (newPassword !== confirm) {
      setError('تکرار گذرواژه مطابقت ندارد')
      return
    }
    setBusy(true)
    try {
      await api.post('/auth/change-password', { old_password: oldPassword, new_password: newPassword })
      setOk(true)
      setTimeout(onClose, 900)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="تغییر گذرواژه" onClose={onClose}>
      {ok ? (
        <Alert kind="success">گذرواژه با موفقیت تغییر کرد؛ نشست‌های دیگر باطل شدند.</Alert>
      ) : (
        <form onSubmit={submit}>
          {error && <Alert kind="error">{error}</Alert>}
          <div className="form-row">
            <label>گذرواژه فعلی</label>
            <input className="input" type="password" value={oldPassword} onChange={(e) => setOld(e.target.value)} />
          </div>
          <div className="form-row">
            <label>گذرواژه جدید (حداقل ۸ نویسه)</label>
            <input className="input" type="password" value={newPassword} onChange={(e) => setNew(e.target.value)} />
          </div>
          <div className="form-row">
            <label>تکرار گذرواژه جدید</label>
            <input className="input" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
          </div>
          <div className="modal-actions">
            <button type="button" className="btn btn-ghost" onClick={onClose}>
              انصراف
            </button>
            <button type="submit" className="btn btn-primary" disabled={busy}>
              ذخیره
            </button>
          </div>
        </form>
      )}
    </Modal>
  )
}
