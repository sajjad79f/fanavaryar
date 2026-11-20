import { useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { Alert } from '../components/ui'
import logo from '../Assets/logo.png'


export default function Login() {
  const { login, user, loading } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const from = (location.state as { from?: string } | null)?.from ?? '/'

  if (!loading && user) {
  return <Navigate to={from} replace />
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await login(username.trim(), password)
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا در ورود')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-page">
      <form className="login-card" onSubmit={submit}>
        <div className="login-brand">
          <img
            src={logo}
            alt="Logo"
            className="login-logo"
            onError={(e) => {
              e.currentTarget.style.display = 'none'
            }}
          />
          <p>سامانه ثبت سوابق واحد IT</p>
        </div>
        {error && <Alert kind="error">{error}</Alert>}
        <div className="form-row">
          <label htmlFor="login-username">نام کاربری</label>
          <input
            id="login-username"
            className="input"
            dir="ltr"
            autoFocus
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
        </div>
        <div className="form-row">
          <label htmlFor="login-password">گذرواژه</label>
          <input
            id="login-password"
            className="input"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <button className="btn btn-primary full" type="submit" disabled={busy}>
          {busy ? 'در حال ورود...' : 'ورود به سامانه'}
        </button>
      </form>
    </div>
  )
}
