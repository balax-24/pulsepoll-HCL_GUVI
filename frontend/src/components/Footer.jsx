import { Link } from 'react-router-dom'
import { LogoMark } from './LogoMark.jsx'

export function Footer() {
  return (
    <footer className="footer-container">
      <div className="footer-content">
        <div className="footer-top-row">
          <div className="footer-brand-section">
            <Link to="/" className="footer-brand-lockup">
              <LogoMark size={28} variant="coral" />
              <span className="footer-brand-name">
                Pulse<span className="brand-wordmark-accent">Poll</span>
              </span>
            </Link>
            <p className="footer-tagline">
              Ask a question. Watch everyone answer.
            </p>
            <p className="footer-subtext">
              Real-time audience polling designed for people and meaningful conversations.
            </p>
          </div>

          <div className="footer-links-section">
            <div className="footer-links-column">
              <span className="footer-column-title">Product</span>
              <Link to="/polls/create" className="footer-link">Create a Poll</Link>
              <Link to="/login" className="footer-link">Creator Sign In</Link>
              <Link to="/register" className="footer-link">Get Started Free</Link>
            </div>
            <div className="footer-links-column">
              <span className="footer-column-title">Architecture</span>
              <span className="footer-static-item">React 19 & Vite</span>
              <span className="footer-static-item">Go / Gin REST API</span>
              <span className="footer-static-item">Redis Atomic Counters</span>
              <span className="footer-static-item">Redis Pub/Sub & WSS</span>
            </div>
          </div>
        </div>

        <div className="footer-bottom-row">
          <div className="footer-meta-text">
            <span>&copy; {new Date().getFullYear()} PulsePoll. Built for the HCL GUVI Selection Assignment.</span>
          </div>
          <div className="footer-tech-badges">
            <span className="tech-badge">Go 1.23</span>
            <span className="tech-badge">MongoDB 7</span>
            <span className="tech-badge">Redis 7</span>
            <span className="tech-badge">WebSockets</span>
          </div>
        </div>
      </div>
    </footer>
  )
}
