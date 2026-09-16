export function PollResults({
  options = [],
  totalVotes = 0,
  userVotedOptionId = null,
  isClosed = false,
  className = '',
}) {
  // Find highest vote count to highlight leading choice
  const maxVotes = options.reduce((max, opt) => {
    const v = opt.count ?? opt.votes ?? 0
    return v > max ? v : max
  }, 0)

  return (
    <div className={`poll-results-container ${className}`}>
      <div className="results-header">
        <div className="results-title-group">
          <div className="results-title-row">
            <h3 className="results-heading">Live Results</h3>
            {isClosed && <span className="badge badge-closed">Voting Closed</span>}
          </div>
          <span className="results-subtext">Realtime audience distribution</span>
        </div>
        <div className="results-meta">
          <div className="total-votes-box">
            <span className="total-votes-count">{totalVotes}</span>
            <span className="total-votes-label">
              {totalVotes === 1 ? 'total vote' : 'total votes'}
            </span>
          </div>
        </div>
      </div>

      <div className="results-options-list">
        {options.map((opt) => {
          const optId = opt.option_id || opt.id
          const count = opt.count ?? opt.votes ?? 0
          const percentage = typeof opt.percentage === 'number'
            ? opt.percentage
            : totalVotes > 0
              ? Math.round((count / totalVotes) * 1000) / 10
              : 0

          const isUserVote = userVotedOptionId === optId
          const isLeading = totalVotes > 0 && count === maxVotes

          return (
            <div
              key={optId}
              className={`result-item ${isUserVote ? 'user-voted-option' : ''} ${
                isLeading ? 'leading-option' : ''
              }`}
            >
              <div className="result-item-header">
                <div className="result-option-info">
                  <span className="result-option-text">{opt.text}</span>
                  {isUserVote && <span className="pill-user-vote">Your Vote</span>}
                  {isLeading && totalVotes > 1 && (
                    <span className="pill-leading">Leading</span>
                  )}
                </div>
                <div className="result-stats">
                  <span className="result-count">
                    {count} {count === 1 ? 'vote' : 'votes'}
                  </span>
                  <span className="result-percentage">
                    {percentage.toFixed(1)}%
                  </span>
                </div>
              </div>

              {/* Animated Progress Bar */}
              <div
                className="progress-track"
                role="progressbar"
                aria-valuenow={percentage}
                aria-valuemin="0"
                aria-valuemax="100"
                aria-label={`${opt.text}: ${percentage.toFixed(1)}%`}
              >
                <div
                  className={`progress-fill ${isLeading ? 'progress-fill-leading' : ''} ${
                    isUserVote ? 'progress-fill-user' : ''
                  }`}
                  style={{ width: `${Math.min(100, Math.max(0, percentage))}%` }}
                />
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
