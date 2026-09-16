/**
 * Centralized API client for PulsePoll
 * Handles environment-based URL configuration, JWT header injection,
 * cookie credential transmission, and structured error normalization.
 */

// Compute API base URL from environment
const rawBase =
  import.meta.env.VITE_API_BASE_URL ||
  import.meta.env.VITE_API_URL ||
  'http://localhost:8080'

// Ensure no trailing slash
const cleanedBase = rawBase.replace(/\/+$/, '')

// Ensure API URL ends with /api
export const API_BASE_URL = cleanedBase.endsWith('/api')
  ? cleanedBase
  : `${cleanedBase}/api`

// Compute WebSocket base URL (e.g. ws://localhost:8080/api or wss://production-domain/api)
// Automatically derives wss:// under https://, or respects explicit VITE_WS_URL
export const WS_BASE_URL =
  (import.meta.env.VITE_WS_URL && import.meta.env.VITE_WS_URL.trim()) ||
  API_BASE_URL.replace(/^http(s)?:\/\//i, (match, s) => (s ? 'wss://' : 'ws://'))

/**
 * Custom API Error class with normalized error code and user-friendly message
 */
export class ApiError extends Error {
  constructor(message, status, code = 'UNKNOWN_ERROR', raw = null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.raw = raw
  }
}

/**
 * Generic request wrapper
 */
export async function request(endpoint, options = {}) {
  const url = `${API_BASE_URL}${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`

  const headers = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
    ...(options.headers || {}),
  }

  // Inject JWT from localStorage if present
  const token = localStorage.getItem('pulsepoll_token')
  if (token && !headers.Authorization) {
    headers.Authorization = `Bearer ${token}`
  }

  const config = {
    ...options,
    headers,
    // Send cookies for voter session tracking (pulsepoll_voter_id)
    credentials: 'include',
  }

  let response
  try {
    response = await fetch(url, config)
  } catch (netErr) {
    throw new ApiError(
      'Unable to connect to the server. Please check your connection and try again.',
      0,
      'NETWORK_ERROR',
      netErr
    )
  }

  // Handle empty responses (e.g. 204 No Content)
  if (response.status === 204) {
    return null
  }

  let data = null
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    try {
      data = await response.json()
    } catch {
      data = null
    }
  }

  if (!response.ok) {
    const status = response.status
    let code = 'HTTP_ERROR'
    let message = 'An unexpected error occurred. Please try again.'

    // Extract structured error from backend envelope
    if (data?.error) {
      if (typeof data.error === 'object') {
        code = data.error.code || code
        message = data.error.message || message
      } else if (typeof data.error === 'string') {
        message = data.error
      }
    } else if (data?.message) {
      message = data.message
    }

    // Friendly overrides for standard status codes if message is generic
    if (status === 401 && (!message || message === 'Unauthorized')) {
      message = 'Your session has expired. Please log in again.'
      code = 'UNAUTHORIZED'
    } else if (status === 403 && (!message || message === 'Forbidden')) {
      message = 'You do not have permission to perform this action.'
      code = 'FORBIDDEN'
    } else if (status === 404 && (!message || message === 'Not Found')) {
      message = 'The requested resource was not found.'
      code = 'NOT_FOUND'
    }

    throw new ApiError(message, status, code, data)
  }

  return data
}
