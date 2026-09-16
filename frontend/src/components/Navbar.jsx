import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'

export function Navbar() {
  const { user, isAuthenticated, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/')
  }

  return (
    <header className="navbar-container">
      <div className="navbar-content">
        <Link to="/" className="navbar-brand">
          <span className="brand-icon-wrapper">
            <span className="brand-pulse-wave" />
            <svg
              className="brand-icon"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
            </svg>
          </span>
          <span className="brand-name">
            Pulse<span className="brand-name-accent">Poll</span>
          </span>
        </Link>

        <nav className="navbar-links" aria-label="Main Navigation">
          <Link to="/" className="nav-link">
            Home
          </Link>
          {isAuthenticated && (
            <>
              <Link to="/dashboard" className="nav-link">
                Dashboard
              </Link>
              <Link to="/polls/create" className="nav-link nav-link-create">
                + Create Poll
              </Link>
            </>
          )}
        </nav>

        <div className="navbar-actions">
          {isAuthenticated ? (
            <div className="user-profile-menu">
              <span className="user-pill" title={user?.email}>
                <span className="user-avatar">{user?.name ? user.name.charAt(0).toUpperCase() : 'U'}</span>
                <span className="user-name">{user?.name || 'Creator'}</span>
              </span>
              <button
                type="button"
                onClick={handleLogout}
                className="btn btn-sm btn-ghost logout-btn"
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
      </div>
    </header>
  )
}
