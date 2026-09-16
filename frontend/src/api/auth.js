import { request } from './client.js'

/**
 * Register a new poll creator
 */
export async function register(name, email, password) {
  return request('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ name, email, password }),
  })
}

/**
 * Login creator with email and password
 * Returns { token, user }
 */
export async function login(email, password) {
  return request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

/**
 * Get current authenticated user details
 * Validates active JWT token
 */
export async function getMe() {
  return request('/auth/me', {
    method: 'GET',
  })
}
