import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import * as authApi from '../api/auth.js'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [token, setToken] = useState(() => localStorage.getItem('pulsepoll_token'))
  const [loading, setLoading] = useState(true)

  // Initialize and verify authentication state on app startup
  useEffect(() => {
    let isMounted = true

    async function checkAuth() {
      const storedToken = localStorage.getItem('pulsepoll_token')
      if (!storedToken) {
        if (isMounted) {
          setUser(null)
          setLoading(false)
        }
        return
      }

      try {
        const data = await authApi.getMe()
        if (isMounted && data?.user) {
          setUser(data.user)
          setToken(storedToken)
        }
      } catch (err) {
        // Token is invalid, expired, or backend unreachable
        console.warn('Authentication token verification failed:', err.message)
        localStorage.removeItem('pulsepoll_token')
        if (isMounted) {
          setUser(null)
          setToken(null)
        }
      } finally {
        if (isMounted) {
          setLoading(false)
        }
      }
    }

    checkAuth()

    return () => {
      isMounted = false
    }
  }, [])

  const login = useCallback(async (email, password) => {
    const data = await authApi.login(email, password)
    if (data?.token && data?.user) {
      localStorage.setItem('pulsepoll_token', data.token)
      setToken(data.token)
      setUser(data.user)
      return data.user
    }
    throw new Error('Invalid response from server during login.')
  }, [])

  const register = useCallback(async (name, email, password) => {
    await authApi.register(name, email, password)
    // After registration, automatically authenticate
    return login(email, password)
  }, [login])

  const logout = useCallback(() => {
    localStorage.removeItem('pulsepoll_token')
    setToken(null)
    setUser(null)
  }, [])

  const value = {
    user,
    token,
    loading,
    isAuthenticated: Boolean(user && token),
    login,
    register,
    logout,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
