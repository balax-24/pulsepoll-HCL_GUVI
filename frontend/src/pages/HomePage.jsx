import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import {
  IconRadio,
  IconShare2,
  IconUsers,
  IconShield,
  IconSliders,
  IconDatabase,
  IconCpu,
  IconServer,
  IconActivity,
  IconArrowRight,
  IconCheck,
} from '../components/Icons.jsx'

export function HomePage() {
  const { isAuthenticated } = useAuth()

  // Interactive Live Demo Simulation State
  const [demoSelected, setDemoSelected] = useState(null)
  const [demoVotes, setDemoVotes] = useState({
    go: 60,
    rust: 40,
    python: 28,
  })
  const [demoLastVoted, setDemoLastVoted] = useState(null)

  // Calculate live demo totals & percentages
  const demoTotal = demoVotes.go + demoVotes.rust + demoVotes.python
  const getPercent = (count) => (demoTotal > 0 ? Math.round((count / demoTotal) * 100) : 0)

  // Gentle subtle realtime tick effect to show live behavior
  useEffect(() => {
    const timer = setInterval(() => {
      // Periodically add a simulated tick to one option
      const choices = ['go', 'rust', 'python']
      const choice = choices[Math.floor(Math.random() * choices.length)]
      setDemoVotes((prev) => ({
        ...prev,
        [choice]: prev[choice] + 1,
      }))
      setDemoLastVoted(choice)
      const clearFlash = setTimeout(() => setDemoLastVoted(null), 1200)
      return () => clearTimeout(clearFlash)
    }, 6000)

    return () => clearInterval(timer)
  }, [])

  const handleDemoVote = (key) => {
    if (demoSelected) return
    setDemoSelected(key)
    setDemoVotes((prev) => ({
      ...prev,
      [key]: prev[key] + 1,
    }))
    setDemoLastVoted(key)
  }

  return (
    <div className="home-page">
      {/* Hero Section */}
      <section className="hero-section">
        <div className="hero-pill-badge">
          <span className="pulse-indicator-dot" />
          <span className="hero-pill-text">Engineered for High-Concurrency Realtime Polling</span>
        </div>

        <h1 className="hero-headline">
          Ask once. <br />
          <span className="headline-gradient">Watch everyone answer. Live.</span>
        </h1>

        <p className="hero-subheadline">
          Create instant interactive polls, distribute a single link to your audience,
          and stream vote tallies live to every screen without refreshing.
        </p>

        <div className="hero-cta-group">
          {isAuthenticated ? (
            <Link to="/polls/create" className="btn btn-lg btn-primary hero-primary-btn">
              <span>Create a Poll</span>
              <IconArrowRight size={18} />
            </Link>
          ) : (
            <>
              <Link to="/register" className="btn btn-lg btn-primary hero-primary-btn">
                <span>Create a Poll</span>
                <IconArrowRight size={18} />
              </Link>
              <a href="#live-demo" className="btn btn-lg btn-secondary hero-secondary-btn">
                <span>Explore Live Demo</span>
              </a>
            </>
          )}
        </div>
      </section>

      {/* Realtime Live Interactive Demo Section */}
      <section id="live-demo" className="demo-showcase-section">
        <div className="demo-showcase-card">
          <div className="demo-header-bar">
            <div className="demo-badge-group">
              <span className="demo-live-badge">
                <span className="live-ping-dot" />
                <span>LIVE NOW</span>
              </span>
              <span className="demo-label-pill">Interactive Product Preview</span>
            </div>
            <span className="demo-hint-text">
              {demoSelected ? '✓ Your sample response recorded' : 'Click any option to test voting'}
            </span>
          </div>

          <div className="demo-content">
            <h2 className="demo-question">What should our team build next?</h2>

            <div className="demo-options-container" role="radiogroup" aria-label="Sample poll choices">
              {/* Option 1: Go */}
              <button
                type="button"
                onClick={() => handleDemoVote('go')}
                disabled={Boolean(demoSelected)}
                className={`demo-option-card ${demoSelected === 'go' ? 'selected' : ''} ${
                  demoLastVoted === 'go' ? 'flash-update' : ''
                }`}
              >
                <div className="demo-option-header">
                  <div className="demo-option-left">
                    <span className="demo-radio-circle">
                      {demoSelected === 'go' && <span className="demo-radio-dot" />}
                    </span>
                    <span className="demo-option-name">Go Microservice Engine</span>
                    {demoSelected === 'go' && <span className="demo-vote-chip">Your Choice</span>}
                  </div>
                  <div className="demo-option-right">
                    <span className="demo-vote-count">{demoVotes.go} votes</span>
                    <span className="demo-percentage">{getPercent(demoVotes.go)}%</span>
                  </div>
                </div>
                <div className="demo-progress-track">
                  <div
                    className="demo-progress-fill demo-fill-primary"
                    style={{ width: `${getPercent(demoVotes.go)}%` }}
                  />
                </div>
              </button>

              {/* Option 2: Rust */}
              <button
                type="button"
                onClick={() => handleDemoVote('rust')}
                disabled={Boolean(demoSelected)}
                className={`demo-option-card ${demoSelected === 'rust' ? 'selected' : ''} ${
                  demoLastVoted === 'rust' ? 'flash-update' : ''
                }`}
              >
                <div className="demo-option-header">
                  <div className="demo-option-left">
                    <span className="demo-radio-circle">
                      {demoSelected === 'rust' && <span className="demo-radio-dot" />}
                    </span>
                    <span className="demo-option-name">Rust High-Performance Worker</span>
                    {demoSelected === 'rust' && <span className="demo-vote-chip">Your Choice</span>}
                  </div>
                  <div className="demo-option-right">
                    <span className="demo-vote-count">{demoVotes.rust} votes</span>
                    <span className="demo-percentage">{getPercent(demoVotes.rust)}%</span>
                  </div>
                </div>
                <div className="demo-progress-track">
                  <div
                    className="demo-progress-fill demo-fill-secondary"
                    style={{ width: `${getPercent(demoVotes.rust)}%` }}
                  />
                </div>
              </button>

              {/* Option 3: Python */}
              <button
                type="button"
                onClick={() => handleDemoVote('python')}
                disabled={Boolean(demoSelected)}
                className={`demo-option-card ${demoSelected === 'python' ? 'selected' : ''} ${
                  demoLastVoted === 'python' ? 'flash-update' : ''
                }`}
              >
                <div className="demo-option-header">
                  <div className="demo-option-left">
                    <span className="demo-radio-circle">
                      {demoSelected === 'python' && <span className="demo-radio-dot" />}
                    </span>
                    <span className="demo-option-name">Python Analytics Pipeline</span>
                    {demoSelected === 'python' && <span className="demo-vote-chip">Your Choice</span>}
                  </div>
                  <div className="demo-option-right">
                    <span className="demo-vote-count">{demoVotes.python} votes</span>
                    <span className="demo-percentage">{getPercent(demoVotes.python)}%</span>
                  </div>
                </div>
                <div className="demo-progress-track">
                  <div
                    className="demo-progress-fill demo-fill-tertiary"
                    style={{ width: `${getPercent(demoVotes.python)}%` }}
                  />
                </div>
              </button>
            </div>

            <div className="demo-footer-bar">
              <div className="demo-live-status">
                <span className="status-live-indicator" />
                <span className="status-live-label">Results updating live</span>
              </div>
              <span className="demo-total-count">
                <strong>{demoTotal}</strong> total responses
              </span>
            </div>
          </div>
        </div>
      </section>

      {/* How it Works: 3 Simple Steps */}
      <section className="workflow-steps-section">
        <div className="section-header-centered">
          <span className="section-overhead-pill">Frictionless Workflow</span>
          <h2 className="section-main-heading">From question to live results in three steps</h2>
        </div>

        <div className="steps-grid">
          <div className="step-card">
            <span className="step-number-tag">01</span>
            <h3 className="step-card-title">Create your poll</h3>
            <p className="step-card-desc">
              Define your question, add up to 10 choices, and optionally schedule an automatic closing deadline.
            </p>
          </div>

          <div className="step-card">
            <span className="step-number-tag">02</span>
            <h3 className="step-card-title">Share the link</h3>
            <p className="step-card-desc">
              Distribute a clean URL or display it on a presentation screen. Audiences can vote instantly with zero login.
            </p>
          </div>

          <div className="step-card">
            <span className="step-number-tag">03</span>
            <h3 className="step-card-title">Watch responses live</h3>
            <p className="step-card-desc">
              Vote tallies, percentages, and leading choices animate in realtime via dedicated WebSocket streaming.
            </p>
          </div>
        </div>
      </section>

      {/* Core Features Grid */}
      <section className="features-grid-section">
        <div className="section-header-centered">
          <span className="section-overhead-pill">Product Capabilities</span>
          <h2 className="section-main-heading">Engineered for audience engagement</h2>
        </div>

        <div className="features-modern-grid">
          <div className="feature-item-card">
            <div className="feature-icon-box">
              <IconRadio size={22} className="text-primary" />
            </div>
            <h3 className="feature-card-heading">Live Results</h3>
            <p className="feature-card-body">
              Persistent WebSocket channels broadcast aggregated vote updates to every connected screen sub-second.
            </p>
          </div>

          <div className="feature-item-card">
            <div className="feature-icon-box">
              <IconActivity size={22} className="text-emerald" />
            </div>
            <h3 className="feature-card-heading">One-Click Sharing</h3>
            <p className="feature-card-body">
              Shareable public links adapt seamlessly to desktop browsers, mobile phones, and projector displays.
            </p>
          </div>

          <div className="feature-item-card">
            <div className="feature-icon-box">
              <IconUsers size={22} className="text-indigo" />
            </div>
            <h3 className="feature-card-heading">Anonymous Audience Voting</h3>
            <p className="feature-card-body">
              Audience members vote with zero barrier to entry. No account registration or app installation required.
            </p>
          </div>

          <div className="feature-item-card">
            <div className="feature-icon-box">
              <IconShield size={22} className="text-amber" />
            </div>
            <h3 className="feature-card-heading">Duplicate Vote Protection</h3>
            <p className="feature-card-body">
              Cryptographic voter tokens and database constraints prevent multiple submissions from the same device.
            </p>
          </div>

          <div className="feature-item-card">
            <div className="feature-icon-box">
              <IconServer size={22} className="text-rose" />
            </div>
            <h3 className="feature-card-heading">Reliable Poll Management</h3>
            <p className="feature-card-body">
              Creators retain full lifecycle control with instantaneous close, question updates, and complete data deletion.
            </p>
          </div>
        </div>
      </section>

      {/* Compact Architecture Section */}
      <section className="architecture-compact-section">
        <div className="architecture-compact-box">
          <div className="architecture-box-header">
            <div className="arch-badge">BUILT FOR REALTIME</div>
            <h2 className="arch-heading">High-concurrency infrastructure</h2>
            <p className="arch-subtext">
              Decoupled architecture separates durable audit persistence from sub-millisecond in-memory tallying.
            </p>
          </div>

          <div className="architecture-stack-grid">
            <div className="arch-stack-item">
              <div className="arch-stack-header">
                <IconDatabase size={20} className="arch-icon text-emerald" />
                <span className="arch-stack-name">MongoDB</span>
              </div>
              <span className="arch-stack-role">Durable vote records</span>
              <p className="arch-stack-desc">
                Immutable audit trail with compound unique index guards against duplicate submissions.
              </p>
            </div>

            <div className="arch-stack-item">
              <div className="arch-stack-header">
                <IconCpu size={20} className="arch-icon text-rose" />
                <span className="arch-stack-name">Redis</span>
              </div>
              <span className="arch-stack-role">Atomic live counters</span>
              <p className="arch-stack-desc">
                High-throughput in-memory tallies and pub/sub message distribution without database write bottlenecks.
              </p>
            </div>

            <div className="arch-stack-item">
              <div className="arch-stack-header">
                <IconRadio size={20} className="arch-icon text-indigo" />
                <span className="arch-stack-name">WebSockets</span>
              </div>
              <span className="arch-stack-role">Instant updates</span>
              <p className="arch-stack-desc">
                Persistent bidirectional connections push real-time broadcasts to all active viewers with automatic reconnection.
              </p>
            </div>

            <div className="arch-stack-item">
              <div className="arch-stack-header">
                <IconServer size={20} className="arch-icon text-cyan" />
                <span className="arch-stack-name">Go & Gin</span>
              </div>
              <span className="arch-stack-role">High-concurrency API</span>
              <p className="arch-stack-desc">
                Lightweight goroutines handle thousands of concurrent voting requests with sub-millisecond execution times.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Bottom CTA Banner */}
      <section className="bottom-cta-banner">
        <div className="bottom-cta-content">
          <h2 className="bottom-cta-title">Ready to engage your audience?</h2>
          <p className="bottom-cta-desc">
            Set up your first live poll in less than a minute. No credit card required.
          </p>
          <div className="bottom-cta-actions">
            <Link to={isAuthenticated ? '/polls/create' : '/register'} className="btn btn-lg btn-primary">
              <span>Get Started Now</span>
              <IconArrowRight size={18} />
            </Link>
          </div>
        </div>
      </section>
    </div>
  )
}
