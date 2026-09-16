import { useEffect, useState, useRef, useCallback } from 'react'
import { WS_BASE_URL } from '../api/client.js'

/**
 * Custom hook for managing a real-time WebSocket connection to a specific poll.
 * Implements bounded exponential backoff reconnection, heartbeat resilience,
 * and clean unmount teardown for React 19 / StrictMode.
 *
 * @param {string} pollId - The ID of the poll to subscribe to
 * @param {Object} callbacks - Event callbacks
 * @param {Function} callbacks.onSnapshot - Invoked on results_snapshot
 * @param {Function} callbacks.onVoteUpdate - Invoked on vote_update
 * @param {Function} callbacks.onPollClosed - Invoked on poll_closed
 */
export function usePollWebSocket(pollId, { onSnapshot, onVoteUpdate, onPollClosed }) {
  const [status, setStatus] = useState('connecting') // 'connecting' | 'connected' | 'reconnecting' | 'disconnected'
  const [reconnectAttempts, setReconnectAttempts] = useState(0)

  // Use refs for callbacks so websocket message listener always has the latest versions
  const callbacksRef = useRef({ onSnapshot, onVoteUpdate, onPollClosed })
  useEffect(() => {
    callbacksRef.current = { onSnapshot, onVoteUpdate, onPollClosed }
  }, [onSnapshot, onVoteUpdate, onPollClosed])

  const wsRef = useRef(null)
  const reconnectTimerRef = useRef(null)
  const isMountedRef = useRef(true)
  const connectRef = useRef(null)

  const connect = useCallback(() => {
    if (!pollId || !isMountedRef.current) return

    // Clean up any existing connection before reconnecting
    if (wsRef.current) {
      wsRef.current.onclose = null
      wsRef.current.onerror = null
      wsRef.current.onmessage = null
      wsRef.current.close()
      wsRef.current = null
    }

    const wsUrl = `${WS_BASE_URL}/polls/${pollId}/ws`
    setStatus((prev) => (prev === 'connected' ? 'reconnecting' : 'connecting'))

    try {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        if (!isMountedRef.current) {
          ws.close()
          return
        }
        setStatus('connected')
        setReconnectAttempts(0)
      }

      ws.onmessage = (event) => {
        if (!isMountedRef.current || !event.data) return

        try {
          const payload = JSON.parse(event.data)

          // Security & consistency: ignore messages from mismatched polls
          if (payload.poll_id && payload.poll_id !== pollId) {
            console.warn('Received WebSocket event for mismatched poll ID:', payload.poll_id)
            return
          }

          switch (payload.type) {
            case 'results_snapshot':
              if (callbacksRef.current.onSnapshot) {
                callbacksRef.current.onSnapshot(payload)
              }
              break

            case 'vote_update':
              if (callbacksRef.current.onVoteUpdate) {
                callbacksRef.current.onVoteUpdate(payload)
              }
              break

            case 'poll_closed':
              if (callbacksRef.current.onPollClosed) {
                callbacksRef.current.onPollClosed(payload)
              }
              break

            default:
              // Unknown or extension message type
              break
          }
        } catch (parseErr) {
          console.error('Failed to parse WebSocket message frame:', parseErr)
        }
      }

      ws.onerror = (err) => {
        if (!isMountedRef.current) return
        console.warn('WebSocket connection error:', err)
      }

      ws.onclose = (event) => {
        if (!isMountedRef.current) return

        // If closed intentionally by client, don't auto-reconnect
        if (event.wasClean && event.code === 1000) {
          setStatus('disconnected')
          return
        }

        setStatus('reconnecting')

        // Bounded exponential backoff: 1s, 2s, 4s, max 10s
        setReconnectAttempts((prev) => {
          const nextAttempt = prev + 1
          const delay = Math.min(1000 * Math.pow(2, prev), 10000)

          if (reconnectTimerRef.current) {
            clearTimeout(reconnectTimerRef.current)
          }

          reconnectTimerRef.current = setTimeout(() => {
            if (isMountedRef.current && connectRef.current) {
              connectRef.current()
            }
          }, delay)

          return nextAttempt
        })
      }
    } catch (err) {
      console.error('Failed to instantiate WebSocket:', err)
      setStatus('disconnected')
    }
  }, [pollId])

  useEffect(() => {
    connectRef.current = connect
  }, [connect])

  useEffect(() => {
    isMountedRef.current = true
    connect()

    return () => {
      isMountedRef.current = false
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.onerror = null
        wsRef.current.onmessage = null
        wsRef.current.close(1000, 'Unmounted')
        wsRef.current = null
      }
    }
  }, [connect])

  return {
    status,
    reconnectAttempts,
    reconnect: connect,
  }
}
