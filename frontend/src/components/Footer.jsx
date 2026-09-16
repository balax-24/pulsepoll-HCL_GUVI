export function Footer() {
  return (
    <footer className="footer-container">
      <div className="footer-content">
        <div className="footer-meta">
          <p className="footer-title">
            <strong>PulsePoll</strong> — High-Concurrency Realtime Polling Platform
          </p>
          <p className="footer-subtitle">
            Engineered with Go, Redis Pub/Sub, MongoDB, and React WebSockets.
          </p>
        </div>
        <div className="footer-badges">
          <span className="tech-badge">Go / Gin</span>
          <span className="tech-badge">Redis Atomic Counters</span>
          <span className="tech-badge">Redis Pub/Sub</span>
          <span className="tech-badge">MongoDB Durable</span>
          <span className="tech-badge">React 19</span>
        </div>
      </div>
    </footer>
  )
}
