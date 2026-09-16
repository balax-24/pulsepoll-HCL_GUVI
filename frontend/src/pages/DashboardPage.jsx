import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { getMyPolls, deletePoll } from '../api/polls.js'
import { PollCard } from '../components/PollCard.jsx'
import { Alert } from '../components/Alert.jsx'
import { IconPlus, IconRefresh, IconBarChart, IconActivity } from '../components/Icons.jsx'
import { useAuth } from '../context/AuthContext.jsx'

export function DashboardPage() {
  const { user } = useAuth()
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
    if (!window.confirm('Are you sure you want to delete this poll? Public viewers will immediately receive a 404.')) {
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
      {/* Workspace Header */}
      <div className="dashboard-header">
        <div className="dashboard-title-group">
          <div className="dashboard-pretitle">
            <span className="dashboard-status-dot" />
            <span>Creator Workspace</span>
          </div>
          <h1 className="dashboard-title">
            {user?.name ? `${user.name}'s Polls` : 'My Polls'}
          </h1>
          <p className="dashboard-subtitle">
            Create questions, distribute shareable links, and monitor live responses.
          </p>
        </div>
        <div className="dashboard-actions">
          <Link to="/polls/create" className="btn btn-primary create-poll-cta">
            <IconPlus size={16} />
            <span>Create Poll</span>
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

      {/* Meaningful Metrics Overview */}
      <div className="metrics-grid">
        <div className="metric-card">
          <div className="metric-header-row">
            <span className="metric-label">Total Polls</span>
            <IconBarChart size={16} className="text-muted" />
          </div>
          <span className="metric-value">{loading ? '—' : totalCount}</span>
          <span className="metric-subtext">Created in your account</span>
        </div>

        <div className="metric-card metric-card-active">
          <div className="metric-header-row">
            <span className="metric-label">Active Polls</span>
            <IconActivity size={16} className="text-mint" />
          </div>
          <span className="metric-value text-mint">{loading ? '—' : activePollsCount}</span>
          <span className="metric-subtext">Accepting live audience votes</span>
        </div>

        <div className="metric-card metric-card-closed">
          <div className="metric-header-row">
            <span className="metric-label">Closed Polls</span>
            <span className="badge-dot dot-gray" />
          </div>
          <span className="metric-value text-muted">{loading ? '—' : closedPollsCount}</span>
          <span className="metric-subtext">Archived final results</span>
        </div>
      </div>

      {/* Filter Tabs & Refresh Bar */}
      <div className="dashboard-filter-bar">
        <div className="filter-tabs" role="tablist" aria-label="Poll Status Filter">
          <button
            type="button"
            className={`filter-tab ${filter === 'all' ? 'active' : ''}`}
            onClick={() => setFilter('all')}
            role="tab"
            aria-selected={filter === 'all'}
          >
            All Polls <span className="tab-badge">{polls.length}</span>
          </button>
          <button
            type="button"
            className={`filter-tab ${filter === 'active' ? 'active' : ''}`}
            onClick={() => setFilter('active')}
            role="tab"
            aria-selected={filter === 'active'}
          >
            Active <span className="tab-badge">{activePollsCount}</span>
          </button>
          <button
            type="button"
            className={`filter-tab ${filter === 'closed' ? 'active' : ''}`}
            onClick={() => setFilter('closed')}
            role="tab"
            aria-selected={filter === 'closed'}
          >
            Closed <span className="tab-badge">{closedPollsCount}</span>
          </button>
        </div>

        <button
          type="button"
          className="btn btn-sm btn-ghost refresh-btn"
          onClick={fetchPolls}
          disabled={loading}
          title="Reload polls list"
          aria-label="Refresh poll list"
        >
          <IconRefresh size={14} className={loading ? 'animate-spin' : ''} />
          <span>Refresh</span>
        </button>
      </div>

      {/* Polls Listing */}
      {loading ? (
        <div className="polls-loading-state" aria-label="Loading polls">
          <div className="loading-spinner" />
          <p className="loading-text">Fetching your live polls...</p>
        </div>
      ) : filteredPolls.length === 0 ? (
        <div className="polls-empty-state">
          <div className="empty-state-visual-wrap">
            <img
              src="/images/empty-state.webp"
              alt="No polls created yet"
              className="empty-state-illustration"
              width="240"
              height="240"
            />
          </div>
          {filter === 'all' ? (
            <>
              <h2 className="empty-state-title">No polls yet. Ask your first question.</h2>
              <p className="empty-state-desc">
                Create a poll in seconds. Share the link with your audience and watch answers stream in live.
              </p>
              <Link to="/polls/create" className="btn btn-primary btn-empty-cta">
                <IconPlus size={16} />
                <span>Create Your First Poll</span>
              </Link>
            </>
          ) : (
            <>
              <h2 className="empty-state-title">No {filter} polls found</h2>
              <p className="empty-state-desc">
                There are currently no polls matching the "{filter}" filter.
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
