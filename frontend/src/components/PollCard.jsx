import { useState } from 'react'
import { Link } from 'react-router-dom'

export function PollCard({ poll, onDelete }) {
  const [copied, setCopied] = useState(false)

  const shareUrl = `${window.location.origin}/polls/${poll.id}`

  const handleCopy = async (e) => {
    e.preventDefault()
    try {
      await navigator.clipboard.writeText(shareUrl)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback
      setCopied(false)
    }
  }

  const isClosed = poll.status === 'closed'
  const isExpired = poll.expires_at && new Date(poll.expires_at) < new Date()
  const createdDate = poll.created_at
    ? new Date(poll.created_at).toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      })
    : 'Recently'

  return (
    <div className={`poll-card ${isClosed ? 'poll-card-closed' : ''}`}>
      <div className="poll-card-header">
        <span className={`badge ${isClosed ? 'badge-closed' : 'badge-active'}`}>
          {isClosed ? 'Closed' : isExpired ? 'Expired' : 'Active'}
        </span>
        <span className="poll-card-date">{createdDate}</span>
      </div>

      <h3 className="poll-card-question">
        <Link to={`/polls/${poll.id}`} className="poll-card-title-link">
          {poll.question}
        </Link>
      </h3>

      <div className="poll-card-options-summary">
        <span className="options-count-pill">
          {poll.options?.length || 0} options
        </span>
      </div>

      <div className="poll-card-actions">
        <button
          type="button"
          onClick={handleCopy}
          className="btn btn-sm btn-secondary copy-link-btn"
          title="Copy public link to clipboard"
        >
          {copied ? '✓ Copied!' : '📋 Copy Link'}
        </button>

        <Link
          to={`/polls/${poll.id}`}
          className="btn btn-sm btn-outline"
        >
          View Live
        </Link>

        <Link
          to={`/polls/${poll.id}/manage`}
          className="btn btn-sm btn-primary"
        >
          Manage
        </Link>

        {onDelete && (
          <button
            type="button"
            onClick={() => onDelete(poll.id)}
            className="btn btn-sm btn-ghost btn-remove-option"
            title="Delete poll"
            aria-label="Delete poll"
          >
            🗑️
          </button>
        )}
      </div>
    </div>
  )
}
