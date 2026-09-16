import { IconAlertTriangle, IconCheckCircle, IconInfo, IconX } from './Icons.jsx'

export function Alert({ type = 'error', message, onClose, className = '' }) {
  if (!message) return null

  const typeStyles = {
    error: 'alert-error',
    warning: 'alert-warning',
    success: 'alert-success',
    info: 'alert-info',
  }

  const icons = {
    error: <IconAlertTriangle size={16} />,
    warning: <IconAlertTriangle size={16} />,
    success: <IconCheckCircle size={16} />,
    info: <IconInfo size={16} />,
  }

  return (
    <div className={`alert ${typeStyles[type] || 'alert-info'} ${className}`} role="alert">
      <span className="alert-icon">{icons[type] || icons.info}</span>
      <span className="alert-message">{message}</span>
      {onClose && (
        <button
          type="button"
          onClick={onClose}
          className="alert-close"
          aria-label="Close notification"
        >
          <IconX size={14} />
        </button>
      )}
    </div>
  )
}
