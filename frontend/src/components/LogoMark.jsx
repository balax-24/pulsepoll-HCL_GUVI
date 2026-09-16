import React from 'react'

/**
 * PulsePoll Official Brand Logo Mark
 * Concept: Speech Bubble (Conversation) + Poll Bars (Voting) + Pulse Wave (Realtime Participation)
 * Rendered as a scalable, clean vector SVG.
 */
export function LogoMark({ size = 32, variant = 'coral', className = '' }) {
  const isNavy = variant === 'navy'

  const bgColor = isNavy ? '#102A43' : '#FF6B5A'
  const barColor = isNavy ? '#B8E8D4' : '#FFFFFF'
  const pulseColor = isNavy ? '#FF6B5A' : '#102A43'

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 32 32"
      width={size}
      height={size}
      fill="none"
      className={`pulsepoll-logo-mark ${className}`}
      aria-hidden="true"
    >
      {/* Speech Bubble Base */}
      <path
        d="M16 3C8.82 3 3 8.37 3 15c0 2.82 1.02 5.41 2.76 7.45L4.2 27.8a0.8 0.8 0 0 0 1 1l5.35-1.56C12.21 28.36 14.04 29 16 29c7.18 0 13-5.37 13-14S23.18 3 16 3z"
        fill={bgColor}
      />
      {/* Voting Poll Bars */}
      <rect x="9.5" y="12.5" width="3" height="8" rx="1.5" fill={barColor} />
      <rect x="14.5" y="9" width="3" height="12" rx="1.5" fill={barColor} />
      <rect x="19.5" y="11" width="3" height="10" rx="1.5" fill={barColor} />
      {/* Realtime Pulse Waveform */}
      <path
        d="M6 15h4.5l2-4 3 8 2.5-6 2 2h8"
        stroke={pulseColor}
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
