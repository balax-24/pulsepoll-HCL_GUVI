import { useState } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { Alert } from '../components/Alert.jsx'
import { LogoMark } from '../components/LogoMark.jsx'
import { IconEye, IconEyeOff } from '../components/Icons.jsx'

export function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const from = location.state?.from?.pathname || '/dashboard'

  const handleSubmit = async (e) => {
    e.preventDefault()
    setFormError('')

    const trimmedEmail = email.trim()
    if (!trimmedEmail) {
      setFormError('Please enter your email address.')
      return
    }

    if (!/\S+@\S+\.\S+/.test(trimmedEmail)) {
      setFormError('Please enter a valid email address.')
      return
    }

    if (!password) {
      setFormError('Please enter your password.')
      return
    }

    setIsSubmitting(true)
    try {
      await login(trimmedEmail, password)
      navigate(from, { replace: true })
    } catch (err) {
      setFormError(err.message || 'Invalid email or password. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="auth-split-wrapper">
      <div className="auth-split-container">
        {/* Left Side: Brand Story & Editorial Illustration */}
        <div className="auth-story-panel">
          <div className="auth-story-header">
            <Link to="/" className="auth-brand-link">
              <LogoMark size={32} variant="coral" />
              <span className="auth-brand-text">
                Pulse<span className="brand-wordmark-accent">Poll</span>
              </span>
            </Link>
          </div>

          <div className="auth-story-body">
            <div className="auth-illustration-wrap">
              <img
                src="/images/auth-visual.webp"
                alt="Audience asking questions and participating in live polls"
                className="auth-editorial-image"
                width="480"
                height="360"
              />
            </div>
            <div className="auth-quote-box">
              <h2 className="auth-quote-title">From question to collective clarity.</h2>
              <p className="auth-quote-text">
                Ask an audience anything. Watch authentic answers, trends, and decisions emerge live in front of you.
              </p>
            </div>
          </div>

          <div className="auth-story-footer">
            <span>&copy; {new Date().getFullYear()} PulsePoll. Built for human participation.</span>
          </div>
        </div>

        {/* Right Side: Authentication Form Card */}
        <div className="auth-form-panel">
          <div className="auth-form-card">
            <div className="auth-header">
              <h1 className="auth-title">Welcome back</h1>
              <p className="auth-subtitle">Sign in to manage your polls and view live results</p>
            </div>

            {formError && (
              <Alert
                type="error"
                message={formError}
                onClose={() => setFormError('')}
                className="auth-alert"
              />
            )}

            <form onSubmit={handleSubmit} className="auth-form" noValidate>
              <div className="form-group">
                <label htmlFor="login-email" className="form-label">
                  Email Address
                </label>
                <input
                  id="login-email"
                  type="email"
                  className="form-input"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="name@example.com"
                  required
                  autoComplete="email"
                  disabled={isSubmitting}
                />
              </div>

              <div className="form-group">
                <div className="label-row">
                  <label htmlFor="login-password" className="form-label">
                    Password
                  </label>
                </div>
                <div className="password-input-wrapper">
                  <input
                    id="login-password"
                    type={showPassword ? 'text' : 'password'}
                    className="form-input password-input"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Enter your password"
                    required
                    autoComplete="current-password"
                    disabled={isSubmitting}
                  />
                  <button
                    type="button"
                    className="password-toggle-btn"
                    onClick={() => setShowPassword((prev) => !prev)}
                    aria-label={showPassword ? 'Hide visibility' : 'Show visibility'}
                    title={showPassword ? 'Hide password' : 'Show password'}
                    tabIndex={-1}
                  >
                    {showPassword ? <IconEyeOff size={16} /> : <IconEye size={16} />}
                  </button>
                </div>
              </div>

              <button
                type="submit"
                className="btn btn-primary btn-block btn-auth-submit"
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <span className="btn-spinner-content">
                    <span className="spinner-inline" />
                    Signing in...
                  </span>
                ) : (
                  'Sign In'
                )}
              </button>
            </form>

            <div className="auth-footer">
              <p className="auth-footer-text">
                Don't have an account yet?{' '}
                <Link to="/register" className="auth-link">
                  Create an account
                </Link>
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
