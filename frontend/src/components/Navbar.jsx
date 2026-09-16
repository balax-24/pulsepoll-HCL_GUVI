import { useState } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { IconActivity, IconPlus, IconX } from './Icons.jsx'

export function Navbar() {
  const { user, isAuthenticated, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  const handleLogout = () => {
    logout()
    setMobileMenuOpen(false)
    navigate('/')
  }

  const isActive = (path) => location.pathname === path

  return (
    <header className="navbar-container">
      <div className="navbar-content">
        {/* Brand */}
        <Link to="/" className="navbar-brand" onClick={() => setMobileMenuOpen(false)}>
          <span className="brand-logo-mark">
            <IconActivity className="brand-pulse-icon" size={18} />
          </span>
          <span className="brand-wordmark">
            Pulse<span className="brand-wordmark-accent">Poll</span>
          </span>
          <span className="brand-environment-badge">Live</span>
        </Link>

        {/* Desktop Navigation Links */}
        <nav className="navbar-links" aria-label="Main Navigation">
          <Link
            to="/"
            className={`nav-link ${isActive('/') ? 'active' : ''}`}
          >
            Home
          </Link>

          {isAuthenticated && (
            <>
              <Link
                to="/dashboard"
                className={`nav-link ${isActive('/dashboard') ? 'active' : ''}`}
              >
                Dashboard
              </Link>
              <Link
                to="/polls/create"
                className={`nav-link nav-link-highlight ${isActive('/polls/create') ? 'active' : ''}`}
              >
                <IconPlus size={14} className="nav-link-icon" />
                <span>Create Poll</span>
              </Link>
            </>
          )}
        </nav>

        {/* Desktop Auth / User Action Area */}
        <div className="navbar-actions">
          {isAuthenticated ? (
            <div className="user-profile-menu">
              <div className="user-info-chip" title={user?.email || 'Logged in user'}>
                <span className="user-avatar-initial">
                  {user?.name ? user.name.charAt(0).toUpperCase() : 'U'}
                </span>
                <span className="user-display-name">{user?.name || 'Creator'}</span>
              </div>
              <button
                type="button"
                onClick={handleLogout}
                className="btn btn-sm btn-ghost logout-btn"
                aria-label="Sign out of PulsePoll"
              >
                Sign Out
              </button>
            </div>
          ) : (
            <div className="auth-buttons-group">
              <Link to="/login" className="btn btn-sm btn-ghost">
                Sign In
              </Link>
              <Link to="/register" className="btn btn-sm btn-primary">
                Get Started
              </Link>
            </div>
          )}
        </div>

        {/* Mobile Navigation Menu Toggle */}
        <button
          type="button"
          className="mobile-menu-toggle"
          onClick={() => setMobileMenuOpen((prev) => !prev)}
          aria-label={mobileMenuOpen ? 'Close mobile menu' : 'Open mobile menu'}
          aria-expanded={mobileMenuOpen}
        >
          {mobileMenuOpen ? (
            <IconX size={20} />
          ) : (
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="3" y1="12" x2="21" y2="12" />
              <line x1="3" y1="6" x2="21" y2="6" />
              <line x1="3" y1="18" x2="21" y2="18" />
            </svg>
          )}
        </button>
      </div>

      {/* Mobile Dropdown Menu */}
      {mobileMenuOpen && (
        <div className="mobile-menu-dropdown">
          <nav className="mobile-nav-links" aria-label="Mobile Navigation">
            <Link
              to="/"
              className={`mobile-nav-link ${isActive('/') ? 'active' : ''}`}
              onClick={() => setMobileMenuOpen(false)}
            >
              Home
            </Link>

            {isAuthenticated ? (
              <>
                <Link
                  to="/dashboard"
                  className={`mobile-nav-link ${isActive('/dashboard') ? 'active' : ''}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Dashboard
                </Link>
                <Link
                  to="/polls/create"
                  className={`mobile-nav-link ${isActive('/polls/create') ? 'active' : ''}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  + Create Poll
                </Link>
                <div className="mobile-user-row">
                  <span className="mobile-user-name">{user?.name || user?.email}</span>
                  <button
                    type="button"
                    onClick={handleLogout}
                    className="btn btn-sm btn-ghost"
                  >
                    Sign Out
                  </button>
                </div>
              </>
            ) : (
              <div className="mobile-auth-row">
                <Link
                  to="/login"
                  className="btn btn-sm btn-outline mobile-btn"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Sign In
                </Link>
                <Link
                  to="/register"
                  className="btn btn-sm btn-primary mobile-btn"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Get Started
                </Link>
              </div>
            )}
          </nav>
        </div>
      )}
    </header>
  )
}
