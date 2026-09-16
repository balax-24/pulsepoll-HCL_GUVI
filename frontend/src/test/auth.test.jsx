import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { LoginPage } from '../pages/LoginPage.jsx'
import { RegisterPage } from '../pages/RegisterPage.jsx'
import { ProtectedRoute } from '../components/ProtectedRoute.jsx'
import { AuthProvider } from '../context/AuthContext.jsx'
import * as authApi from '../api/auth.js'

vi.mock('../api/auth.js', () => ({
  login: vi.fn(),
  register: vi.fn(),
  getMe: vi.fn(),
}))

describe('Authentication & Protected Routes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('validates required fields on LoginPage', async () => {
    render(
      <MemoryRouter>
        <AuthProvider>
          <LoginPage />
        </AuthProvider>
      </MemoryRouter>
    )

    const submitBtn = screen.getByRole('button', { name: /Sign In/i })
    fireEvent.click(submitBtn)

    expect(screen.getByText(/Please enter your email address/i)).toBeInTheDocument()

    const emailInput = screen.getByLabelText(/Email Address/i)
    fireEvent.change(emailInput, { target: { value: 'invalid-email' } })
    fireEvent.click(submitBtn)

    expect(screen.getByText(/Please enter a valid email address/i)).toBeInTheDocument()
  })

  it('validates password length on RegisterPage', async () => {
    render(
      <MemoryRouter>
        <AuthProvider>
          <RegisterPage />
        </AuthProvider>
      </MemoryRouter>
    )

    const nameInput = screen.getByLabelText(/Full Name/i)
    const emailInput = screen.getByLabelText(/Email Address/i)
    const passwordInput = screen.getByLabelText(/Password/i)
    const submitBtn = screen.getByRole('button', { name: /Create Account/i })

    fireEvent.change(nameInput, { target: { value: 'Alice Smith' } })
    fireEvent.change(emailInput, { target: { value: 'alice@example.com' } })
    fireEvent.change(passwordInput, { target: { value: 'short' } })
    fireEvent.click(submitBtn)

    expect(screen.getByText(/Password must be at least 8 characters/i)).toBeInTheDocument()
  })

  it('redirects unauthenticated users in ProtectedRoute to /login', () => {
    // getMe fails so user remains unauthenticated
    authApi.getMe.mockRejectedValue(new Error('Unauthorized'))

    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<div>Login Screen</div>} />
            <Route
              path="/dashboard"
              element={
                <ProtectedRoute>
                  <div>Secret Dashboard Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    )

    // Initially or after auth check completes, unauthenticated goes to login screen
    // Since localStorage has no token, loading finishes immediately
    expect(screen.getByText('Login Screen')).toBeInTheDocument()
    expect(screen.queryByText('Secret Dashboard Content')).not.toBeInTheDocument()
  })
})
