import { useState } from 'react'
import { Link } from 'react-router-dom'
import { IconCopy, IconCheck, IconExternalLink, IconEdit, IconTrash } from './Icons.jsx'

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
        <div className="poll-card-badge-group">
          <span className={`badge ${isClosed ? 'badge-closed' : isExpired ? 'badge-expired' : 'badge-active'}`}>
            <span className="badge-dot" />
            {isClosed ? 'Closed' : isExpired ? 'Expired' : 'Active'}
          </span>
          <span className="options-count-pill">
            {poll.options?.length || 0} options
          </span>
        </div>
        <span className="poll-card-date">{createdDate}</span>
      </div>

      <h3 className="poll-card-question">
        <Link to={`/polls/${poll.id}`} className="poll-card-title-link">
          {poll.question}
        </Link>
      </h3>

      <div className="poll-card-actions">
        <button
          type="button"
          onClick={handleCopy}
          className="btn btn-sm btn-secondary copy-link-btn"
          title="Copy public link to clipboard"
          aria-label="Copy public poll link"
        >
          {copied ? (
            <>
              <IconCheck size={14} className="text-emerald" />
              <span>Copied!</span>
            </>
          ) : (
            <>
              <IconCopy size={14} />
              <span>Copy Link</span>
            </>
          )}
        </button>

        <Link
          to={`/polls/${poll.id}`}
          className="btn btn-sm btn-outline"
          title="Open live public voting view"
        >
          <IconExternalLink size={14} />
          <span>View Live</span>
        </Link>

        <Link
          to={`/polls/${poll.id}/manage`}
          className="btn btn-sm btn-primary"
          title="Manage poll settings and closure"
        >
          <IconEdit size={14} />
          <span>Manage</span>
        </Link>

        {onDelete && (
          <button
            type="button"
            onClick={() => onDelete(poll.id)}
            className="btn btn-sm btn-ghost btn-danger-ghost delete-poll-btn"
            title="Delete this poll"
            aria-label="Delete poll"
          >
            <IconTrash size={14} />
          </button>
        )}
      </div>
    </div>
  )
}
