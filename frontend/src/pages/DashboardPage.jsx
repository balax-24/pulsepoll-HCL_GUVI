import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { getMyPolls, deletePoll } from '../api/polls.js'
import { PollCard } from '../components/PollCard.jsx'
import { Alert } from '../components/Alert.jsx'

export function DashboardPage() {
  const [polls, setPolls] = useState([])
  const [totalCount, setTotalCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('all') // 'all' | 'active' | 'closed'
  const [actionSuccess, setActionSuccess] = useState('')

  const fetchPolls = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await getMyPolls(1, 50)
      setPolls(data?.polls || [])
      setTotalCount(data?.total_count || 0)
    } catch (err) {
      setError(err.message || 'Failed to load your polls. Please check connection and try again.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchPolls()
  }, [fetchPolls])

  const handleDelete = async (pollId) => {
    if (!window.confirm('Are you sure you want to delete this poll? Public viewers will no longer be able to access it.')) {
      return
    }

    try {
      await deletePoll(pollId)
      setPolls((prev) => prev.filter((p) => p.id !== pollId))
      setTotalCount((prev) => Math.max(0, prev - 1))
      setActionSuccess('Poll was deleted successfully.')
      setTimeout(() => setActionSuccess(''), 3000)
    } catch (err) {
      setError(err.message || 'Failed to delete poll.')
    }
  }

  const activePollsCount = polls.filter((p) => p.status === 'active').length
  const closedPollsCount = polls.filter((p) => p.status === 'closed').length

  const filteredPolls = polls.filter((p) => {
    if (filter === 'active') return p.status === 'active'
    if (filter === 'closed') return p.status === 'closed'
    return true
  })

  return (
    <div className="dashboard-container">
      <div className="dashboard-header">
        <div className="dashboard-title-group">
          <h1 className="dashboard-title">Creator Dashboard</h1>
          <p className="dashboard-subtitle">Monitor real-time participation and manage your polls</p>
        </div>
        <div className="dashboard-actions">
          <Link to="/polls/create" className="btn btn-primary">
            <span className="btn-icon">+</span> Create Poll
          </Link>
        </div>
      </div>

      {actionSuccess && (
        <Alert
          type="success"
          message={actionSuccess}
          onClose={() => setActionSuccess('')}
          className="dashboard-alert"
        />
      )}

      {error && (
        <Alert
          type="error"
          message={error}
          onClose={() => setError('')}
          className="dashboard-alert"
        />
      )}

      {/* Metrics Row */}
      <div className="metrics-grid">
        <div className="metric-card">
          <span className="metric-label">Total Created</span>
          <span className="metric-value">{loading ? '—' : totalCount}</span>
          <span className="metric-subtext">Polls in your account</span>
        </div>
        <div className="metric-card metric-card-active">
          <span className="metric-label">Active Polls</span>
          <span className="metric-value text-emerald">{loading ? '—' : activePollsCount}</span>
          <span className="metric-subtext">Accepting live responses</span>
        </div>
        <div className="metric-card metric-card-closed">
          <span className="metric-label">Closed Polls</span>
          <span className="metric-value text-muted">{loading ? '—' : closedPollsCount}</span>
          <span className="metric-subtext">Archived / final results</span>
        </div>
      </div>

      {/* Filter Tabs */}
      <div className="dashboard-filter-bar">
        <div className="filter-tabs" role="tablist">
          <button
            type="button"
            className={`filter-tab ${filter === 'all' ? 'active' : ''}`}
            onClick={() => setFilter('all')}
            role="tab"
            aria-selected={filter === 'all'}
          >
            All Polls ({polls.length})
          </button>
          <button
            type="button"
            className={`filter-tab ${filter === 'active' ? 'active' : ''}`}
            onClick={() => setFilter('active')}
            role="tab"
            aria-selected={filter === 'active'}
          >
            Active ({activePollsCount})
          </button>
          <button
            type="button"
            className={`filter-tab ${filter === 'closed' ? 'active' : ''}`}
            onClick={() => setFilter('closed')}
            role="tab"
            aria-selected={filter === 'closed'}
          >
            Closed ({closedPollsCount})
          </button>
        </div>

        <button
          type="button"
          className="btn btn-sm btn-ghost refresh-btn"
          onClick={fetchPolls}
          disabled={loading}
          title="Reload polls list"
        >
          ↻ Refresh
        </button>
      </div>

      {/* Polls Listing */}
      {loading ? (
        <div className="polls-loading-state" aria-label="Loading polls">
          <div className="loading-spinner" />
          <p className="loading-text">Loading your polls...</p>
        </div>
      ) : filteredPolls.length === 0 ? (
        <div className="polls-empty-state">
          <div className="empty-state-icon">📊</div>
          {filter === 'all' ? (
            <>
              <h2 className="empty-state-title">You haven't created any polls yet.</h2>
              <p className="empty-state-desc">
                Engage your audience with live questions and watch instant responses roll in in real-time.
              </p>
              <Link to="/polls/create" className="btn btn-primary btn-lg">
                Create Your First Poll
              </Link>
            </>
          ) : (
            <>
              <h2 className="empty-state-title">No {filter} polls found.</h2>
              <p className="empty-state-desc">
                Try switching filters to view all your polls.
              </p>
              <button
                type="button"
                className="btn btn-outline"
                onClick={() => setFilter('all')}
              >
                Show All Polls
              </button>
            </>
          )}
        </div>
      ) : (
        <div className="polls-grid">
          {filteredPolls.map((poll) => (
            <PollCard
              key={poll.id}
              poll={poll}
              onDelete={handleDelete}
            />
          ))}
        </div>
      )}
    </div>
  )
}
