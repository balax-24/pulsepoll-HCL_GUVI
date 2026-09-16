export function LiveBadge({ status = 'connected', className = '' }) {
  const configs = {
    connected: {
      label: 'Live Updates',
      colorClass: 'live-connected',
      dotClass: 'dot-pulse-green',
    },
    connecting: {
      label: 'Connecting...',
      colorClass: 'live-connecting',
      dotClass: 'dot-pulse-blue',
    },
    reconnecting: {
      label: 'Reconnecting...',
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
    <div className={`live-badge ${current.colorClass} ${className}`} title={`Status: ${status}`}>
      <span className={`status-dot ${current.dotClass}`} />
      <span className="status-label">{current.label}</span>
    </div>
  )
}
