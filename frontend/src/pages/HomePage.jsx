import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'

export function HomePage() {
  const { isAuthenticated } = useAuth()

  return (
    <div className="home-page-container">
      {/* Hero Section */}
      <section className="hero-section">
        <div className="hero-badge">
          <span className="hero-badge-pulse" />
          Realtime Polling Engine Powered by Redis & Go
        </div>

        <h1 className="hero-title">
          Live Audience Polling <br />
          <span className="hero-title-gradient">Without Page Refreshes</span>
        </h1>

        <p className="hero-description">
          Create interactive polls in seconds, share instant public links with your audience,
          and watch vote tallies animate live across all screens via high-throughput
          Go WebSockets and Redis Pub/Sub broadcasting.
        </p>

        <div className="hero-actions">
          {isAuthenticated ? (
            <Link to="/dashboard" className="btn btn-lg btn-primary hero-btn">
              Go to Your Dashboard →
            </Link>
          ) : (
            <>
              <Link to="/register" className="btn btn-lg btn-primary hero-btn">
                Create Free Poll →
              </Link>
              <Link to="/login" className="btn btn-lg btn-secondary hero-btn">
                Creator Sign In
              </Link>
            </>
          )}
        </div>
      </section>

      {/* Feature Grid */}
      <section className="features-section">
        <div className="feature-card">
          <div className="feature-icon">⚡</div>
          <h3 className="feature-title">Atomic Redis Live Counters</h3>
          <p className="feature-desc">
            Vote aggregations mutate in-memory using atomic <code>HINCRBY</code> operations,
            preventing lost updates even during massive concurrent spikes.
          </p>
        </div>

        <div className="feature-card">
          <div className="feature-icon">📡</div>
          <h3 className="feature-title">Realtime Pub/Sub Broadcasting</h3>
          <p className="feature-desc">
            No repeated REST polling loops or intervals. Redis Pub/Sub routes events directly
            into Go WebSocket hubs for instant sub-millisecond client updates.
          </p>
        </div>

        <div className="feature-card">
          <div className="feature-icon">🛡️</div>
          <h3 className="feature-title">Durable Audit & Duplicate Guard</h3>
          <p className="feature-desc">
            Every vote is durably logged in MongoDB with unique compound index constraints on
            <code>(poll_id, voter_id)</code> to prevent duplicate submissions.
          </p>
        </div>

        <div className="feature-card">
          <div className="feature-icon">🌐</div>
          <h3 className="feature-title">Frictionless Public Voting</h3>
          <p className="feature-desc">
            Public viewers participate immediately with no signup required. Sessions are
            tracked anonymously via secure HTTP cookies and headers.
          </p>
        </div>
      </section>

      {/* Architecture Overview Banner */}
      <section className="architecture-banner">
        <h2 className="banner-title">Architectural Separation of Concerns</h2>
        <div className="architecture-flow-diagram">
          <div className="flow-step">
            <span className="step-tag">Step 1</span>
            <strong>Public Vote</strong>
            <span>POST /api/polls/:id/vote</span>
          </div>
          <div className="flow-arrow">→</div>
          <div className="flow-step">
            <span className="step-tag">Step 2</span>
            <strong>MongoDB Write</strong>
            <span>Durable vote audit record</span>
          </div>
          <div className="flow-arrow">→</div>
          <div className="flow-step">
            <span className="step-tag">Step 3</span>
            <strong>Redis HINCRBY</strong>
            <span>Atomic count mutation</span>
          </div>
          <div className="flow-arrow">→</div>
          <div className="flow-step">
            <span className="step-tag">Step 4</span>
            <strong>Redis Pub/Sub</strong>
            <span>poll:&#123;id&#125;:updates</span>
          </div>
          <div className="flow-arrow">→</div>
          <div className="flow-step">
            <span className="step-tag">Step 5</span>
            <strong>WebSocket Hub</strong>
            <span>Broadcasts to all browsers</span>
          </div>
        </div>
      </section>
    </div>
  )
}
