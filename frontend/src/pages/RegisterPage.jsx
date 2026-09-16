import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { Alert } from '../components/Alert.jsx'
import { LogoMark } from '../components/LogoMark.jsx'
import { IconEye, IconEyeOff } from '../components/Icons.jsx'

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
              <h2 className="auth-quote-title">Engage your audience in real time.</h2>
              <p className="auth-quote-text">
                Create beautiful live polls in seconds. Watch everyone participate freely from their phones with zero app downloads.
              </p>
            </div>
          </div>

          <div className="auth-story-footer">
            <span>&copy; {new Date().getFullYear()} PulsePoll. Built for human participation.</span>
          </div>
        </div>

        {/* Right Side: Registration Form Card */}
        <div className="auth-form-panel">
          <div className="auth-form-card">
            <div className="auth-header">
              <h1 className="auth-title">Create your account</h1>
              <p className="auth-subtitle">Start building live polls and gathering real-time answers</p>
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
                <span className="input-hint">Minimum 8 characters</span>
              </div>

              <button
                type="submit"
                className="btn btn-primary btn-block btn-auth-submit"
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <span className="btn-spinner-content">
                    <span className="spinner-inline" />
                    Creating account...
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
      </div>
    </div>
  )
}
