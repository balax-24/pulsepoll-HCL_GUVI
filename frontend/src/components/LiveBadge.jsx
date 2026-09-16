export function LiveBadge({ status = 'connected', className = '' }) {
  const configs = {
    connected: {
      label: 'Live Updates',
      colorClass: 'live-connected',
      dotClass: 'dot-pulse-emerald',
      symbol: '●',
    },
    connecting: {
      label: 'Connecting...',
      colorClass: 'live-connecting',
      dotClass: 'dot-pulse-indigo',
      symbol: '◌',
    },
    reconnecting: {
      label: 'Reconnecting...',
      colorClass: 'live-reconnecting',
      dotClass: 'dot-pulse-amber',
      symbol: '↻',
    },
    disconnected: {
      label: 'Offline',
      colorClass: 'live-disconnected',
      dotClass: 'dot-gray',
      symbol: '○',
    },
  }

  const current = configs[status] || configs.disconnected

  return (
    <div
      className={`live-badge ${current.colorClass} ${className}`}
      title={`Connection: ${status}`}
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
