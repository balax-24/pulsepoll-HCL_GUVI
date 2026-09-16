import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { Alert } from '../components/Alert.jsx'
import { IconActivity, IconEye, IconEyeOff } from '../components/Icons.jsx'

export function RegisterPage() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const { register } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e) => {
    e.preventDefault()
    setFormError('')

    const trimmedName = name.trim()
    const trimmedEmail = email.trim()

    if (!trimmedName || trimmedName.length < 2) {
      setFormError('Name must be at least 2 characters.')
      return
    }

    if (!trimmedEmail) {
      setFormError('Please enter your email address.')
      return
    }

    if (!/\S+@\S+\.\S+/.test(trimmedEmail)) {
      setFormError('Please enter a valid email address.')
      return
    }

    if (!password || password.length < 8) {
      setFormError('Password must be at least 8 characters long.')
      return
    }

    setIsSubmitting(true)
    try {
      await register(trimmedName, trimmedEmail, password)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setFormError(err.message || 'Registration failed. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="auth-page-container">
      <div className="auth-card">
        {/* Top Branding & Header */}
        <div className="auth-header">
          <div className="auth-brand-badge">
            <IconActivity size={20} className="auth-pulse-icon" />
          </div>
          <h1 className="auth-title">Create Account</h1>
          <p className="auth-subtitle">Start creating live, interactive polls in seconds</p>
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
            <label htmlFor="register-name" className="form-label">
              Full Name
            </label>
            <input
              id="register-name"
              type="text"
              className="form-input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Alex Morgan"
              required
              autoComplete="name"
              disabled={isSubmitting}
            />
          </div>

          <div className="form-group">
            <label htmlFor="register-email" className="form-label">
              Email Address
            </label>
            <input
              id="register-email"
              type="email"
              className="form-input"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="alex@company.com"
              required
              autoComplete="email"
              disabled={isSubmitting}
            />
          </div>

          <div className="form-group">
            <label htmlFor="register-password" className="form-label">
              Password
            </label>
            <div className="password-input-wrapper">
              <input
                id="register-password"
                type={showPassword ? 'text' : 'password'}
                className="form-input password-input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="At least 8 characters"
                required
                autoComplete="new-password"
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
            <span className="input-hint">Minimum 8 characters with at least one letter</span>
          </div>

          <button
            type="submit"
            className="btn btn-primary btn-block btn-auth-submit"
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <span className="btn-spinner-content">
                <span className="spinner-inline" />
                Creating Account...
              </span>
            ) : (
              'Create Account'
            )}
          </button>
        </form>

        <div className="auth-footer">
          <p className="auth-footer-text">
            Already have an account?{' '}
            <Link to="/login" className="auth-link">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
