export function LiveBadge({ status = 'connected', className = '' }) {
  const configs = {
    connected: {
      label: 'Live',
      colorClass: 'live-connected',
      dotClass: 'dot-pulse-mint',
    },
    connecting: {
      label: 'Connecting',
      colorClass: 'live-connecting',
      dotClass: 'dot-pulse-peach',
    },
    reconnecting: {
      label: 'Reconnecting',
      colorClass: 'live-reconnecting',
      dotClass: 'dot-pulse-amber',
    },
    disconnected: {
      label: 'Offline',
      colorClass: 'live-disconnected',
      dotClass: 'dot-gray',
    },
  }

  const current = configs[status] || configs.disconnected

  return (
    <div
      className={`live-badge ${current.colorClass} ${className}`}
      title={`Realtime status: ${status}`}
      role="status"
      aria-live="polite"
    >
      <span className={`status-dot ${current.dotClass}`}>
        <span className="dot-inner" />
        <span className="dot-ring" />
      </span>
      <span className="status-label">{current.label}</span>
    </div>
  )
}
