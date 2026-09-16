import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { getPublicPoll, closePoll, deletePoll, updatePoll } from '../api/polls.js'
import { getResults } from '../api/votes.js'
import { usePollWebSocket } from '../hooks/usePollWebSocket.js'
import { PollResults } from '../components/PollResults.jsx'
import { LiveBadge } from '../components/LiveBadge.jsx'
import { Alert } from '../components/Alert.jsx'
import {
  IconCopy,
  IconCheck,
  IconEdit,
  IconTrash,
  IconExternalLink,
  IconLock,
  IconAlertTriangle,
  IconSliders,
  IconClock,
} from '../components/Icons.jsx'

export function ManagePollPage() {
  const { id: pollId } = useParams()
  const navigate = useNavigate()

  const [poll, setPoll] = useState(null)
  const [results, setResults] = useState({ options: [], totalVotes: 0 })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionSuccess, setActionSuccess] = useState('')
  const [isProcessing, setIsProcessing] = useState(false)
  const [copied, setCopied] = useState(false)

  // Question editing state
  const [isEditingQuestion, setIsEditingQuestion] = useState(false)
  const [editedQuestion, setEditedQuestion] = useState('')

  const fetchPollDetails = useCallback(async () => {
    if (!pollId) return
    setLoading(true)
    setError('')
    try {
      const pollData = await getPublicPoll(pollId)
      const actualPoll = pollData?.poll || pollData
      setPoll(actualPoll)
      setEditedQuestion(actualPoll.question)

      try {
        const resultsData = await getResults(pollId)
        if (resultsData?.results) {
          setResults({
            options: resultsData.results,
            totalVotes: resultsData.total_votes || 0,
          })
        }
      } catch {
        // Fallback options
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
      if (err.status === 403 || err.code === 'FORBIDDEN') {
        setError('You do not have permission to manage this poll.')
      } else if (err.status === 404) {
        setError('Poll not found or has been deleted.')
      } else {
        setError(err.message || 'Failed to load poll details.')
      }
    } finally {
      setLoading(false)
    }
  }, [pollId])

  useEffect(() => {
    fetchPollDetails()
  }, [fetchPollDetails])

  // Real-time synchronization
  const handleSnapshot = useCallback((payload) => {
    if (payload.poll_id && payload.poll_id !== pollId) return
    if (Array.isArray(payload.options)) {
      setResults({
        options: payload.options,
        totalVotes: payload.total_votes || 0,
      })
    }
  }, [pollId])

  const handleVoteUpdate = useCallback((payload) => {
    if (payload.poll_id && payload.poll_id !== pollId) return
    setResults((prev) => {
      const newTotal = payload.total_votes ?? (prev.totalVotes + 1)
      const updatedOptions = prev.options.map((opt) => {
        const isTarget = opt.option_id === payload.option_id || opt.id === payload.option_id
        const newCount = isTarget ? payload.count : (opt.count ?? opt.votes ?? 0)
        const newPercentage = newTotal > 0 ? Math.round((newCount / newTotal) * 1000) / 10 : 0
        return {
          ...opt,
          count: newCount,
          votes: newCount,
          percentage: newPercentage,
        }
      })
      return { options: updatedOptions, totalVotes: newTotal }
    })
  }, [pollId])

  const handlePollClosedWs = useCallback((payload) => {
    if (payload.poll_id && payload.poll_id !== pollId) return
    setPoll((prev) => (prev ? { ...prev, status: 'closed' } : prev))
  }, [pollId])

  const { status: wsStatus } = usePollWebSocket(pollId, {
    onSnapshot: handleSnapshot,
    onVoteUpdate: handleVoteUpdate,
    onPollClosed: handlePollClosedWs,
  })

  // Creator Actions
  const handleClosePoll = async () => {
    if (!window.confirm('Are you sure you want to close this poll? Audience members will immediately no longer be able to cast votes.')) {
      return
    }

    setIsProcessing(true)
    setError('')
    try {
      const response = await closePoll(pollId)
      const updated = response?.poll || response
      setPoll((prev) => ({ ...prev, status: 'closed', ...updated }))
      setActionSuccess('Poll has been successfully closed. No further votes will be accepted.')
      setTimeout(() => setActionSuccess(''), 4000)
    } catch (err) {
      if (err.status === 403 || err.code === 'FORBIDDEN') {
        setError('Forbidden: Only the creator of this poll can close it.')
      } else {
        setError(err.message || 'Failed to close poll.')
      }
    } finally {
      setIsProcessing(false)
    }
  }

  const handleDeletePoll = async () => {
    if (!window.confirm('Are you sure you want to delete this poll? This action cannot be undone and public access will be disabled.')) {
      return
    }

    setIsProcessing(true)
    setError('')
    try {
      await deletePoll(pollId)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      if (err.status === 403 || err.code === 'FORBIDDEN') {
        setError('Forbidden: Only the creator of this poll can delete it.')
      } else {
        setError(err.message || 'Failed to delete poll.')
      }
      setIsProcessing(false)
    }
  }

  const handleSaveQuestion = async (e) => {
    e.preventDefault()
    const trimmed = editedQuestion.trim()
    if (trimmed.length < 3 || trimmed.length > 255) {
      setError('Question must be between 3 and 255 characters.')
      return
    }

    setIsProcessing(true)
    setError('')
    try {
      const response = await updatePoll(pollId, { question: trimmed })
      const updated = response?.poll || response
      setPoll((prev) => ({ ...prev, question: updated.question || trimmed }))
      setIsEditingQuestion(false)
      setActionSuccess('Question updated successfully.')
      setTimeout(() => setActionSuccess(''), 3000)
    } catch (err) {
      setError(err.message || 'Failed to update question.')
    } finally {
      setIsProcessing(false)
    }
  }

  const publicUrl = `${window.location.origin}/polls/${pollId}`
  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(publicUrl)
      setCopied(true)
      setTimeout(() => setCopied(false), 2500)
    } catch {
      setCopied(false)
    }
  }

  if (loading) {
    return (
      <div className="manage-poll-loading" aria-label="Loading poll manager">
        <div className="loading-spinner" />
        <p className="loading-text">Loading creator management console...</p>
      </div>
    )
  }

  if (error && !poll) {
    return (
      <div className="poll-error-card">
        <div className="error-icon-wrapper">
          <IconLock size={32} className="text-amber" />
        </div>
        <h2 className="error-title">Access Restricted</h2>
        <p className="error-desc">{error}</p>
        <Link to="/dashboard" className="btn btn-primary mt-4">
          Return to Dashboard
        </Link>
      </div>
    )
  }

  const isClosed = poll?.status === 'closed'

  return (
    <div className="manage-poll-container">
      {/* Header / Console Top Bar */}
      <div className="manage-header-card">
        <div className="manage-top-row">
          <div className="manage-status-group">
            <LiveBadge status={wsStatus} />
            <span className={`badge ${isClosed ? 'badge-closed' : 'badge-active'}`}>
              <span className={`badge-dot ${isClosed ? 'dot-gray' : 'dot-emerald'}`} />
              <span>{isClosed ? 'Closed' : 'Active'}</span>
            </span>
          </div>

          <div className="manage-nav-actions">
            <Link to={`/polls/${pollId}`} className="btn btn-sm btn-outline">
              <IconExternalLink size={14} />
              <span>Open Public View</span>
            </Link>
            <Link to="/dashboard" className="btn btn-sm btn-ghost">
              Back to Dashboard
            </Link>
          </div>
        </div>

        {/* Question Title & Inline Edit Form */}
        {isEditingQuestion ? (
          <form onSubmit={handleSaveQuestion} className="edit-question-form">
            <input
              type="text"
              value={editedQuestion}
              onChange={(e) => setEditedQuestion(e.target.value)}
              className="form-input mb-2 form-input-lg"
              disabled={isProcessing}
              required
            />
            <div className="edit-buttons-row">
              <button
                type="submit"
                className="btn btn-sm btn-primary"
                disabled={isProcessing}
              >
                Save
              </button>
              <button
                type="button"
                className="btn btn-sm btn-ghost"
                onClick={() => {
                  setEditedQuestion(poll?.question || '')
                  setIsEditingQuestion(false)
                }}
                disabled={isProcessing}
              >
                Cancel
              </button>
            </div>
          </form>
        ) : (
          <div className="question-display-row">
            <h1 className="manage-question">{poll?.question}</h1>
            {!isClosed && (
              <button
                type="button"
                onClick={() => setIsEditingQuestion(true)}
                className="btn btn-sm btn-ghost edit-btn"
                title="Edit poll question"
                aria-label="Edit question"
              >
                <IconEdit size={14} />
                <span>Edit</span>
              </button>
            )}
          </div>
        )}

        <div className="manage-meta-row">
          <span className="meta-item">
            Poll ID: <code className="meta-code">{pollId}</code>
          </span>
          <span className="meta-item">
            Created: {poll?.created_at ? new Date(poll.created_at).toLocaleString() : 'Recently'}
          </span>
          {poll?.expires_at && (
            <span className="meta-item meta-deadline">
              <IconClock size={12} className="meta-inline-icon" />
              Deadline: {new Date(poll.expires_at).toLocaleString()}
            </span>
          )}
        </div>
      </div>

      {actionSuccess && (
        <Alert
          type="success"
          message={actionSuccess}
          onClose={() => setActionSuccess('')}
          className="my-3"
        />
      )}

      {error && (
        <Alert
          type="error"
          message={error}
          onClose={() => setError('')}
          className="my-3"
        />
      )}

      {/* Share Section */}
      <div className="manage-share-card">
        <h3 className="section-heading">Shareable Public Link</h3>
        <p className="section-subtext">Distribute this link to participants to collect votes in real time:</p>
        <div className="share-input-row">
          <input
            type="text"
            readOnly
            value={publicUrl}
            className="form-input share-url-input"
            aria-label="Public Poll URL"
          />
          <button
            type="button"
            onClick={handleCopyLink}
            className="btn btn-secondary copy-action-btn"
            aria-label="Copy Link"
          >
            {copied ? (
              <>
                <IconCheck size={14} className="text-emerald" />
                <span>✓ Link Copied!</span>
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

      {/* Grid: Results & Controls */}
      <div className="manage-grid">
        {/* Real-time Results View */}
        <div className="manage-results-column">
          <PollResults
            options={results.options}
            totalVotes={results.totalVotes}
            isClosed={isClosed}
          />
        </div>

        {/* Creator Control Actions */}
        <div className="manage-controls-column">
          <div className="control-card">
            <h3 className="control-card-title">Poll Controls</h3>
            <p className="control-card-desc">
              Manage lifecycle and access permissions for this poll.
            </p>

            <div className="control-actions-stack">
              {!isClosed ? (
                <div className="control-item">
                  <div className="control-item-info">
                    <strong>Close Poll</strong>
                    <p>Prevents any new votes from being accepted. Current results become final.</p>
                  </div>
                  <button
                    type="button"
                    onClick={handleClosePoll}
                    className="btn btn-warning"
                    disabled={isProcessing}
                  >
                    {isProcessing ? 'Closing...' : 'Close Poll'}
                  </button>
                </div>
              ) : (
                <div className="control-item closed-state-info">
                  <div className="closed-state-header">
                    <span className="badge badge-closed">Poll is Closed</span>
                  </div>
                  <p className="closed-state-desc">This poll is archived and no longer accepting votes.</p>
                </div>
              )}

              <div className="control-item control-item-danger">
                <div className="control-item-info">
                  <strong>Delete Poll</strong>
                  <p>Permanently removes this poll. Public viewers will receive a 404.</p>
                </div>
                <button
                  type="button"
                  onClick={handleDeletePoll}
                  className="btn btn-danger delete-btn"
                  disabled={isProcessing}
                >
                  <IconTrash size={14} />
                  <span>{isProcessing ? 'Deleting...' : 'Delete Poll'}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
