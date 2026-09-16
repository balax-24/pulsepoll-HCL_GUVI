export function Alert({ type = 'error', message, onClose, className = '' }) {
  if (!message) return null

  const typeStyles = {
    error: 'alert-error',
    warning: 'alert-warning',
    success: 'alert-success',
    info: 'alert-info',
  }

  const icons = {
    error: '⚠️',
    warning: '⚡',
    success: '✓',
    info: 'ℹ️',
  }

  return (
    <div className={`alert ${typeStyles[type] || 'alert-info'} ${className}`} role="alert">
      <span className="alert-icon">{icons[type]}</span>
      <span className="alert-message">{message}</span>
      {onClose && (
        <button
          type="button"
          onClick={onClose}
          className="alert-close"
          aria-label="Close notification"
        >
          ×
        </button>
      )}
    </div>
  )
}
