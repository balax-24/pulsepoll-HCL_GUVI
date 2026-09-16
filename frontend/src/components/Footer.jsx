export function Footer() {
  return (
    <footer className="footer-container">
      <div className="footer-content">
        <div className="footer-brand-section">
          <div className="footer-brand-title">
            <span className="footer-logo-dot" />
            <span className="footer-brand-name">Pulse<strong>Poll</strong></span>
          </div>
          <p className="footer-tagline">Realtime audience polling.</p>
        </div>

        <div className="footer-tech-section">
          <span className="footer-tech-label">Engineered with</span>
          <div className="footer-badges">
            <span className="tech-badge">Go Gin</span>
            <span className="tech-badge">Redis Pub/Sub</span>
            <span className="tech-badge">MongoDB</span>
            <span className="tech-badge">WebSockets</span>
          </div>
        </div>
      </div>
    </footer>
  )
}
