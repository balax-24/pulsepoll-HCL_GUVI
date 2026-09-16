import { request } from './client.js'

/**
 * Create a new poll (Authenticated creator)
 * @param {Object} pollData - { question, options: string[], expires_at?: string }
 */
export async function createPoll(pollData) {
  return request('/polls', {
    method: 'POST',
    body: JSON.stringify(pollData),
  })
}

/**
 * Get public poll details by ID (Public audience, no auth required)
 * @param {string} pollId
 */
export async function getPublicPoll(pollId) {
  return request(`/polls/${pollId}`, {
    method: 'GET',
  })
}

/**
 * Get current creator's polls (Authenticated)
 * @param {number} page
 * @param {number} limit
 */
export async function getMyPolls(page = 1, limit = 20) {
  return request(`/my/polls?page=${page}&limit=${limit}`, {
    method: 'GET',
  })
}

/**
 * Update question or options text (Creator only)
 * @param {string} pollId
 * @param {Object} updateData - { question?, options?: Array<{id, text}> }
 */
export async function updatePoll(pollId, updateData) {
  return request(`/polls/${pollId}`, {
    method: 'PATCH',
    body: JSON.stringify(updateData),
  })
}

/**
 * Close poll immediately (Creator only)
 * @param {string} pollId
 */
export async function closePoll(pollId) {
  return request(`/polls/${pollId}/close`, {
    method: 'POST',
  })
}

/**
 * Soft-delete poll (Creator only)
 * @param {string} pollId
 */
export async function deletePoll(pollId) {
  return request(`/polls/${pollId}`, {
    method: 'DELETE',
  })
}
