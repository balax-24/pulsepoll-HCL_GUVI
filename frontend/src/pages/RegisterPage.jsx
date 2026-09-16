import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { Alert } from '../components/Alert.jsx'

export function RegisterPage() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
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
        <div className="auth-header">
          <div className="auth-icon-badge">
            <svg
              className="auth-header-icon"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
              <circle cx="8.5" cy="7" r="4" />
              <line x1="20" y1="8" x2="20" y2="14" />
              <line x1="23" y1="11" x2="17" y2="11" />
            </svg>
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
              placeholder="alex@example.com"
              required
              autoComplete="email"
              disabled={isSubmitting}
            />
          </div>

          <div className="form-group">
            <label htmlFor="register-password" className="form-label">
              Password
            </label>
            <input
              id="register-password"
              type="password"
              className="form-input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="At least 8 characters"
              required
              autoComplete="new-password"
              disabled={isSubmitting}
            />
            <span className="input-hint">Minimum 8 characters with at least one letter</span>
          </div>

          <button
            type="submit"
            className="btn btn-primary btn-block"
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
