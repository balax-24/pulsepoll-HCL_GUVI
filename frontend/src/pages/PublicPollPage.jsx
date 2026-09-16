import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getPublicPoll } from '../api/polls.js'
import { getResults, castVote } from '../api/votes.js'
import { usePollWebSocket } from '../hooks/usePollWebSocket.js'
import { PollResults } from '../components/PollResults.jsx'
import { LiveBadge } from '../components/LiveBadge.jsx'
import { Alert } from '../components/Alert.jsx'
import { IconCopy, IconCheck, IconCheckCircle, IconAlertTriangle, IconClock } from '../components/Icons.jsx'

export function PublicPollPage() {
  const { id: pollId } = useParams()

  const [poll, setPoll] = useState(null)
  const [results, setResults] = useState({ options: [], totalVotes: 0 })
  const [selectedOptionId, setSelectedOptionId] = useState('')
  const [userVotedOptionId, setUserVotedOptionId] = useState(() => {
    return localStorage.getItem(`pulsepoll_voted_${pollId}`) || null
  })
  const [hasVoted, setHasVoted] = useState(() => {
    return Boolean(localStorage.getItem(`pulsepoll_voted_${pollId}`))
  })
  const [isClosed, setIsClosed] = useState(false)
  const [isExpired, setIsExpired] = useState(false)

  const [loadingPoll, setLoadingPoll] = useState(true)
  const [isVoting, setIsVoting] = useState(false)
  const [pollError, setPollError] = useState('')
  const [voteError, setVoteError] = useState('')
  const [voteSuccess, setVoteSuccess] = useState('')
  const [copied, setCopied] = useState(false)

  // 1. Initial REST data fetching
  const fetchInitialData = useCallback(async () => {
    if (!pollId) return
    setLoadingPoll(true)
    setPollError('')

    try {
      // Fetch public poll metadata
      const pollData = await getPublicPoll(pollId)
      const actualPoll = pollData?.poll || pollData
      setPoll(actualPoll)

      const closed = actualPoll.status === 'closed'
      setIsClosed(closed)

      if (actualPoll.expires_at) {
        const expired = new Date(actualPoll.expires_at) <= new Date()
        setIsExpired(expired)
      }

      // Fetch initial live results snapshot
      try {
        const resultsData = await getResults(pollId)
        if (resultsData?.results) {
          setResults({
            options: resultsData.results,
            totalVotes: resultsData.total_votes || 0,
          })
        }
      } catch (resultsErr) {
        console.warn('Initial results fetch fallback:', resultsErr.message)
        // If results endpoint failed, initialize empty results from poll options
        if (actualPoll.options) {
          setResults({
            options: actualPoll.options.map((opt) => ({
              option_id: opt.id,
              text: opt.text,
              count: 0,
              votes: 0,
              percentage: 0,
            })),
            totalVotes: 0,
          })
        }
      }
    } catch (err) {
      if (err.status === 404 || err.message?.toLowerCase().includes('not found')) {
        setPollError('Poll not found. It may have been deleted or never existed.')
      } else {
        setPollError(err.message || 'Unable to connect to the server. Please try again.')
      }
    } finally {
      setLoadingPoll(false)
    }
  }, [pollId])

  useEffect(() => {
    fetchInitialData()
  }, [fetchInitialData])

  // 2. Real-time WebSocket subscriptions
  const handleSnapshot = useCallback(
    (payload) => {
      if (payload.poll_id && payload.poll_id !== pollId) return

      if (Array.isArray(payload.options)) {
        setResults({
          options: payload.options,
          totalVotes: payload.total_votes || 0,
        })
      }
    },
    [pollId]
  )

  const handleVoteUpdate = useCallback(
    (payload) => {
      if (payload.poll_id && payload.poll_id !== pollId) return

      setResults((prev) => {
        const newTotal = payload.total_votes ?? (prev.totalVotes + 1)
        const updatedOptions = prev.options.map((opt) => {
          const isTarget = opt.option_id === payload.option_id || opt.id === payload.option_id
          const newCount = isTarget ? payload.count : (opt.count ?? opt.votes ?? 0)
          const newPercentage =
            newTotal > 0 ? Math.round((newCount / newTotal) * 1000) / 10 : 0

          return {
            ...opt,
            count: newCount,
            votes: newCount,
            percentage: newPercentage,
          }
        })

        return {
          options: updatedOptions,
          totalVotes: newTotal,
        }
      })
    },
    [pollId]
  )

  const handlePollClosed = useCallback(
    (payload) => {
      if (payload.poll_id && payload.poll_id !== pollId) return
      setIsClosed(true)
      setVoteError('Voting is now closed for this poll.')
    },
    [pollId]
  )

  const { status: wsStatus } = usePollWebSocket(pollId, {
    onSnapshot: handleSnapshot,
    onVoteUpdate: handleVoteUpdate,
    onPollClosed: handlePollClosed,
  })

  // 3. Voting submission handler
  const handleVoteSubmit = async (e) => {
    e.preventDefault()
    setVoteError('')
    setVoteSuccess('')

    if (!selectedOptionId) {
      setVoteError('Please select an option before casting your vote.')
      return
    }

    if (isClosed) {
      setVoteError('Voting is closed.')
      return
    }

    if (isExpired) {
      setVoteError('This poll has expired.')
      return
    }

    setIsVoting(true)
    try {
      await castVote(pollId, selectedOptionId)
      setHasVoted(true)
      setUserVotedOptionId(selectedOptionId)
      localStorage.setItem(`pulsepoll_voted_${pollId}`, selectedOptionId)
      setVoteSuccess('Your vote has been recorded!')
    } catch (err) {
      const msg = err.message || ''
      const code = err.code || ''

      if (code === 'DUPLICATE_VOTE' || msg.toLowerCase().includes('already voted')) {
        setHasVoted(true)
        setUserVotedOptionId(selectedOptionId)
        localStorage.setItem(`pulsepoll_voted_${pollId}`, selectedOptionId)
        setVoteError("You've already voted in this poll.")
      } else if (code === 'POLL_CLOSED' || msg.toLowerCase().includes('closed')) {
        setIsClosed(true)
        setVoteError('Voting is closed.')
      } else if (code === 'POLL_EXPIRED' || msg.toLowerCase().includes('expired')) {
        setIsExpired(true)
        setVoteError('This poll has expired.')
      } else {
        setVoteError(msg || 'Unable to submit vote. Please try again.')
      }
    } finally {
      setIsVoting(false)
    }
  }

  // 4. Share link helper
  const shareUrl = window.location.href
  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl)
      setCopied(true)
      setTimeout(() => setCopied(false), 2500)
    } catch {
      setCopied(false)
    }
  }

  if (loadingPoll) {
    return (
      <div className="public-poll-loading" aria-label="Loading poll">
        <div className="loading-spinner" />
        <p className="loading-text">Loading poll and live results...</p>
      </div>
    )
  }

  if (pollError) {
    return (
      <div className="poll-error-card">
        <div className="error-icon-wrapper">
          <IconAlertTriangle size={32} className="text-amber" />
        </div>
        <h2 className="error-title">Unable to Open Poll</h2>
        <p className="error-desc">{pollError}</p>
        <Link to="/" className="btn btn-primary mt-4">
          Return to Home
        </Link>
      </div>
    )
  }

  const pollOptions = poll?.options || []
  const votingDisabled = hasVoted || isClosed || isExpired || isVoting

  return (
    <div className="public-poll-page">
      {/* Poll Header Bar */}
      <div className="public-poll-header-card">
        <div className="poll-top-bar">
          <div className="poll-status-badges">
            <LiveBadge status={wsStatus} />
            {isClosed ? (
              <span className="badge badge-closed">
                <span className="badge-dot dot-gray" />
                <span>Closed</span>
              </span>
            ) : isExpired ? (
              <span className="badge badge-expired">
                <span className="badge-dot dot-amber" />
                <span>Expired</span>
              </span>
            ) : (
              <span className="badge badge-active">
                <span className="badge-dot dot-emerald" />
                <span>Active</span>
              </span>
            )}
          </div>

          <div className="poll-share-actions">
            <button
              type="button"
              onClick={handleCopyLink}
              className="btn btn-sm btn-secondary copy-btn"
              title="Copy shareable link"
              aria-label="Copy Link"
            >
              {copied ? (
                <>
                  <IconCheck size={14} className="text-emerald" />
                  <span>Link Copied!</span>
                </>
              ) : (
                <>
                  <IconCopy size={14} />
                  <span>Copy Link</span>
                </>
              )}
            </button>
          </div>
        </div>

        <h1 className="public-poll-question">{poll?.question}</h1>

        <div className="poll-meta-row">
          <span className="meta-item">
            Created{' '}
            {poll?.created_at
              ? new Date(poll.created_at).toLocaleDateString(undefined, {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric',
                })
              : 'recently'}
          </span>
          {poll?.expires_at && (
            <span className="meta-item meta-item-deadline">
              <IconClock size={12} className="meta-inline-icon" />
              <span>
                Deadline:{' '}
                {new Date(poll.expires_at).toLocaleString(undefined, {
                  month: 'short',
                  day: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </span>
            </span>
          )}
        </div>
      </div>

      {/* Notifications */}
      {voteSuccess && (
        <Alert
          type="success"
          message={voteSuccess}
          onClose={() => setVoteSuccess('')}
          className="my-3"
        />
      )}

      {voteError && (
        <Alert
          type="error"
          message={voteError}
          onClose={() => setVoteError('')}
          className="my-3"
        />
      )}

      {/* Closed / Expired State Banners */}
      {isClosed && (
        <div className="poll-notice-banner banner-closed">
          <span className="notice-badge">POLL CLOSED</span>
          <span className="notice-text">This poll is no longer accepting responses. Final results are displayed below.</span>
        </div>
      )}

      {!isClosed && isExpired && (
        <div className="poll-notice-banner banner-expired">
          <span className="notice-badge">POLL ENDED</span>
          <span className="notice-text">Voting ended because the poll reached its scheduled deadline.</span>
        </div>
      )}

      {/* Main Grid: Voting Card + Results Card */}
      <div className="poll-layout-grid">
        {/* Voting Panel */}
        <div className="poll-voting-card">
          <div className="card-header">
            <h2 className="card-heading">
              {hasVoted ? 'Your Selection' : isClosed ? 'Voting Closed' : isExpired ? 'Poll Expired' : 'Cast Your Vote'}
            </h2>
            <p className="card-subtext">
              {hasVoted
                ? 'Your response is logged. Watch the live distribution update in real time.'
                : isClosed
                ? 'This poll is no longer accepting new responses.'
                : isExpired
                ? 'The deadline for this poll has passed.'
                : 'Select one option below and submit to record your anonymous vote.'}
            </p>
          </div>

          <form onSubmit={handleVoteSubmit} className="voting-form">
            <div className="options-selection-list" role="radiogroup" aria-label="Poll choices">
              {pollOptions.map((opt) => {
                const optId = opt.id || opt.option_id
                const isSelected = selectedOptionId === optId
                const isUserChoice = userVotedOptionId === optId

                return (
                  <label
                    key={optId}
                    htmlFor={`poll-opt-${optId}`}
                    className={`option-choice-label ${isSelected ? 'selected' : ''} ${
                      isUserChoice ? 'voted-choice' : ''
                    } ${votingDisabled ? 'disabled' : ''}`}
                  >
                    <input
                      id={`poll-opt-${optId}`}
                      type="radio"
                      name="poll-choice"
                      value={optId}
                      checked={isSelected}
                      onChange={() => setSelectedOptionId(optId)}
                      disabled={votingDisabled}
                      className="option-radio-input"
                    />
                    <div className="option-choice-content">
                      <span className="option-choice-text">{opt.text}</span>
                      {isUserChoice && <span className="pill-choice-voted">Selected</span>}
                    </div>
                  </label>
                )
              })}
            </div>

            {!hasVoted && !isClosed && !isExpired && (
              <button
                type="submit"
                className="btn btn-primary btn-block btn-vote-submit"
                disabled={isVoting || !selectedOptionId}
              >
                {isVoting ? (
                  <span className="btn-spinner-content">
                    <span className="spinner-inline" />
                    Submitting Vote...
                  </span>
                ) : (
                  'Submit Vote'
                )}
              </button>
            )}

            {hasVoted && (
              <div className="voted-confirmation-box">
                <IconCheckCircle size={18} className="text-emerald" />
                <span className="voted-message">
                  Your vote has been counted and broadcast live to all active viewers.
                </span>
              </div>
            )}
          </form>
        </div>

        {/* Real-time Results Panel */}
        <div className="poll-results-card">
          <PollResults
            options={results.options}
            totalVotes={results.totalVotes}
            userVotedOptionId={userVotedOptionId}
            isClosed={isClosed || isExpired}
          />
        </div>
      </div>
    </div>
  )
}
