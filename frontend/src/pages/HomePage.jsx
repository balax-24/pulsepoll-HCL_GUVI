import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { LogoMark } from '../components/LogoMark.jsx'
import {
  IconArrowRight,
  IconCheck,
  IconRadio,
  IconUsers,
  IconShield,
  IconSparkles,
} from '../components/Icons.jsx'

export function HomePage() {
  const { isAuthenticated } = useAuth()

  // Interactive Live Demo Simulation State
  const [demoSelected, setDemoSelected] = useState(null)
  const [demoVotes, setDemoVotes] = useState({
    hybrid: 52,
    remote: 38,
    office: 16,
  })
  const [demoLastVoted, setDemoLastVoted] = useState(null)

  // Calculate live demo totals & percentages
  const demoTotal = demoVotes.hybrid + demoVotes.remote + demoVotes.office
  const getPercent = (count) => (demoTotal > 0 ? Math.round((count / demoTotal) * 100) : 0)

  // Gentle subtle realtime tick effect to show live behavior
  useEffect(() => {
    const timer = setInterval(() => {
      const choices = ['hybrid', 'remote', 'office']
      const choice = choices[Math.floor(Math.random() * choices.length)]
      setDemoVotes((prev) => ({
        ...prev,
        [choice]: prev[choice] + 1,
      }))
      setDemoLastVoted(choice)
      const clearFlash = setTimeout(() => setDemoLastVoted(null), 1200)
      return () => clearTimeout(clearFlash)
    }, 6500)

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
      {/* 1. Hero Section — Human, Welcoming, Editorial */}
      <section className="hero-section">
        <div className="hero-container">
          <div className="hero-text-column">
            <div className="hero-kicker-badge">
              <span className="kicker-pulse-dot" />
              <span>Real-time audience polling</span>
            </div>

            <h1 className="hero-main-title">
              Ask a question. <br />
              <span className="title-highlight">Watch everyone answer.</span>
            </h1>

            <p className="hero-lead-text">
              Turn any room, meeting, or online audience into an active conversation.
              Create a poll in seconds, share one link, and watch responses stream in live.
            </p>

            <div className="hero-cta-row">
              {isAuthenticated ? (
                <Link to="/polls/create" className="btn btn-lg btn-primary hero-btn-coral">
                  <span>Create a Poll</span>
                  <IconArrowRight size={18} />
                </Link>
              ) : (
                <Link to="/register" className="btn btn-lg btn-primary hero-btn-coral">
                  <span>Create a Poll</span>
                  <IconArrowRight size={18} />
                </Link>
              )}
              <a href="#how-it-works" className="btn btn-lg btn-ghost hero-btn-ghost">
                <span>See how it works</span>
              </a>
            </div>

            {/* Quick Trust Highlights */}
            <div className="hero-trust-row">
              <div className="trust-item">
                <IconCheck size={16} className="trust-icon" />
                <span>No app downloads</span>
              </div>
              <div className="trust-item">
                <IconCheck size={16} className="trust-icon" />
                <span>No sign-up for voters</span>
              </div>
              <div className="trust-item">
                <IconCheck size={16} className="trust-icon" />
                <span>Instant live sync</span>
              </div>
            </div>
          </div>

          {/* Hero Visual Column */}
          <div className="hero-visual-column">
            <div className="hero-visual-frame">
              <img
                src="/images/hero-visual.webp"
                alt="Diverse people participating together in a live PulsePoll session"
                className="hero-editorial-image"
                width="640"
                height="360"
                loading="eager"
              />
              <div className="hero-visual-floating-card">
                <div className="floating-card-icon">
                  <LogoMark size={24} variant="coral" />
                </div>
                <div className="floating-card-text">
                  <span className="floating-card-title">Live Participation</span>
                  <span className="floating-card-desc">Responses update live without refresh</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* 2. Visual Storytelling: 01 CREATE · 02 SHARE · 03 WATCH LIVE */}
      <section id="how-it-works" className="storytelling-section">
        <div className="section-intro">
          <span className="section-eyebrow">How PulsePoll Works</span>
          <h2 className="section-headline">Simple for you. Effortless for your audience.</h2>
          <p className="section-subheadline">
            Three natural steps from asking a question to seeing collective clarity.
          </p>
        </div>

        <div className="story-campaign-list">
          {/* Step 01: CREATE */}
          <div className="story-row story-row-normal">
            <div className="story-image-wrap">
              <img
                src="/images/workflow-create.webp"
                alt="Creator typing a question into PulsePoll interface"
                className="story-image"
                loading="lazy"
                width="500"
                height="375"
              />
            </div>
            <div className="story-content-wrap">
              <span className="story-step-index">01 &mdash; CREATE</span>
              <h3 className="story-title">Turn a question into a poll.</h3>
              <p className="story-description">
                Type what you want to ask. Add 2 to 10 choices. Customize options or set an automatic closing deadline.
                No complex forms, templates, or setup wizards.
              </p>
              <div className="story-feature-list">
                <span className="story-pill">Clean option editor</span>
                <span className="story-pill">Optional deadlines</span>
                <span className="story-pill">Instant preview</span>
              </div>
            </div>
          </div>

          {/* Step 02: SHARE */}
          <div className="story-row story-row-reverse">
            <div className="story-content-wrap">
              <span className="story-step-index">02 &mdash; SHARE</span>
              <h3 className="story-title">Send one link to everyone.</h3>
              <p className="story-description">
                Copy the share link or project it onto a presentation screen.
                Audience members tap the link on their phones or laptops and vote immediately.
                No account required, no passwords, zero barrier to participation.
              </p>
              <div className="story-feature-list">
                <span className="story-pill">One-click copy</span>
                <span className="story-pill">Zero login for voters</span>
                <span className="story-pill">Any mobile browser</span>
              </div>
            </div>
            <div className="story-image-wrap">
              <img
                src="/images/workflow-share.webp"
                alt="Phone displaying shareable poll link connecting people"
                className="story-image"
                loading="lazy"
                width="500"
                height="375"
              />
            </div>
          </div>

          {/* Step 03: WATCH LIVE */}
          <div className="story-row story-row-normal">
            <div className="story-image-wrap">
              <img
                src="/images/workflow-watch.webp"
                alt="Screen with animated live voting bars and audience reacting"
                className="story-image"
                loading="lazy"
                width="500"
                height="375"
              />
            </div>
            <div className="story-content-wrap">
              <span className="story-step-index">03 &mdash; WATCH LIVE</span>
              <h3 className="story-title">See responses appear as they happen.</h3>
              <p className="story-description">
                As votes roll in, the results bar smoothly animates across all connected screens at once.
                Percentages, counts, and leading choices adjust in real time over persistent WebSockets.
              </p>
              <div className="story-feature-list">
                <span className="story-pill">Smooth CSS transitions</span>
                <span className="story-pill">Live vote counts</span>
                <span className="story-pill">No manual refreshing</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* 3. Interactive Live Product Preview Card */}
      <section className="demo-preview-section">
        <div className="demo-container">
          <div className="demo-preview-header">
            <div className="demo-eyebrow-group">
              <span className="demo-status-dot-mint" />
              <span className="demo-preview-tag">Interactive Product Preview</span>
            </div>
            <h2 className="demo-headline">Experience the feel of a live poll right now</h2>
            <p className="demo-subheadline">
              Tap an option below to see how responses calculate and animate for audiences in real time.
            </p>
          </div>

          <div className="demo-card-surface">
            <div className="demo-card-top">
              <div className="demo-card-meta">
                <span className="demo-badge-live">● Live</span>
                <span className="demo-badge-type">Team Decision Poll</span>
              </div>
              <span className="demo-action-hint">
                {demoSelected ? '✓ Your sample vote was cast' : 'Select an option to test'}
              </span>
            </div>

            <h3 className="demo-poll-question">
              Which working model creates the best team energy for 2026?
            </h3>

            <div className="demo-options-list" role="radiogroup" aria-label="Interactive poll sample options">
              {/* Option 1 */}
              <button
                type="button"
                onClick={() => handleDemoVote('hybrid')}
                disabled={Boolean(demoSelected)}
                className={`demo-option-item ${demoSelected === 'hybrid' ? 'voted' : ''} ${
                  demoLastVoted === 'hybrid' ? 'flash-tick' : ''
                }`}
              >
                <div className="demo-option-row">
                  <div className="demo-option-label">
                    <span className="demo-option-number">01</span>
                    <span className="demo-option-text">Hybrid (2–3 flexible days in office)</span>
                    {demoSelected === 'hybrid' && <span className="demo-vote-pill">Your Vote</span>}
                  </div>
                  <div className="demo-option-stats">
                    <span className="demo-count-val">{demoVotes.hybrid} votes</span>
                    <span className="demo-pct-val">{getPercent(demoVotes.hybrid)}%</span>
                  </div>
                </div>
                <div className="demo-bar-track">
                  <div
                    className="demo-bar-fill demo-bar-coral"
                    style={{ width: `${getPercent(demoVotes.hybrid)}%` }}
                  />
                </div>
              </button>

              {/* Option 2 */}
              <button
                type="button"
                onClick={() => handleDemoVote('remote')}
                disabled={Boolean(demoSelected)}
                className={`demo-option-item ${demoSelected === 'remote' ? 'voted' : ''} ${
                  demoLastVoted === 'remote' ? 'flash-tick' : ''
                }`}
              >
                <div className="demo-option-row">
                  <div className="demo-option-label">
                    <span className="demo-option-number">02</span>
                    <span className="demo-option-text">Fully Remote (Work from anywhere)</span>
                    {demoSelected === 'remote' && <span className="demo-vote-pill">Your Vote</span>}
                  </div>
                  <div className="demo-option-stats">
                    <span className="demo-count-val">{demoVotes.remote} votes</span>
                    <span className="demo-pct-val">{getPercent(demoVotes.remote)}%</span>
                  </div>
                </div>
                <div className="demo-bar-track">
                  <div
                    className="demo-bar-fill demo-bar-mint"
                    style={{ width: `${getPercent(demoVotes.remote)}%` }}
                  />
                </div>
              </button>

              {/* Option 3 */}
              <button
                type="button"
                onClick={() => handleDemoVote('office')}
                disabled={Boolean(demoSelected)}
                className={`demo-option-item ${demoSelected === 'office' ? 'voted' : ''} ${
                  demoLastVoted === 'office' ? 'flash-tick' : ''
                }`}
              >
                <div className="demo-option-row">
                  <div className="demo-option-label">
                    <span className="demo-option-number">03</span>
                    <span className="demo-option-text">In-Person First (Collaborative studio)</span>
                    {demoSelected === 'office' && <span className="demo-vote-pill">Your Vote</span>}
                  </div>
                  <div className="demo-option-stats">
                    <span className="demo-count-val">{demoVotes.office} votes</span>
                    <span className="demo-pct-val">{getPercent(demoVotes.office)}%</span>
                  </div>
                </div>
                <div className="demo-bar-track">
                  <div
                    className="demo-bar-fill demo-bar-navy"
                    style={{ width: `${getPercent(demoVotes.office)}%` }}
                  />
                </div>
              </button>
            </div>

            <div className="demo-card-footer">
              <div className="demo-footer-info">
                <span className="demo-live-dot" />
                <span>Simulating live responses from audience</span>
              </div>
              <span className="demo-footer-total">
                <strong>{demoTotal}</strong> participants voted
              </span>
            </div>
          </div>
        </div>
      </section>

      {/* 4. Human Product Values Section */}
      <section className="values-section">
        <div className="values-container">
          <div className="values-header">
            <span className="section-eyebrow">Designed for Real Participation</span>
            <h2 className="section-headline">Why people love using PulsePoll</h2>
          </div>

          <div className="values-grid">
            <div className="value-card">
              <div className="value-icon-box value-icon-coral">
                <IconUsers size={22} />
              </div>
              <h3 className="value-title">Zero Friction for Voters</h3>
              <p className="value-text">
                Your audience joins via phone, tablet, or laptop in one tap. No accounts, no email forms, no app installs.
              </p>
            </div>

            <div className="value-card">
              <div className="value-icon-box value-icon-mint">
                <IconRadio size={22} />
              </div>
              <h3 className="value-title">Instant Collective Energy</h3>
              <p className="value-text">
                Live result bars reveal audience opinions immediately. Perfect for company standups, classroom Q&As, and conference stages.
              </p>
            </div>

            <div className="value-card">
              <div className="value-icon-box value-icon-peach">
                <IconShield size={22} />
              </div>
              <h3 className="value-title">Fair Single-Vote Integrity</h3>
              <p className="value-text">
                Smart session tokens and database-level unique constraints prevent duplicate submissions so your results stay authentic.
              </p>
            </div>

            <div className="value-card">
              <div className="value-icon-box value-icon-navy">
                <IconSparkles size={22} />
              </div>
              <h3 className="value-title">Dual-Layer Reliability</h3>
              <p className="value-text">
                Permanent audit records in MongoDB combine with fast atomic Redis counters to ensure your data is always safe and responsive.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* 5. Bottom Call to Action Banner — Deep Navy with Warm Coral */}
      <section className="cta-banner-section">
        <div className="cta-banner-card">
          <div className="cta-banner-content">
            <span className="cta-kicker">Get Started In Under 60 Seconds</span>
            <h2 className="cta-title">Ready to hear what your audience thinks?</h2>
            <p className="cta-subtext">
              Create your first live poll for free. Ask a question, share the link, and see real-time answers unfold.
            </p>
            <div className="cta-action-row">
              <Link to="/polls/create" className="btn btn-lg btn-primary hero-btn-coral">
                <span>Create a Live Poll Now</span>
                <IconArrowRight size={18} />
              </Link>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
