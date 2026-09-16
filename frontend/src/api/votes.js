import { request } from './client.js'

/**
 * Cast a vote on a public poll
 * @param {string} pollId
 * @param {string} optionId
 */
export async function castVote(pollId, optionId) {
  return request(`/polls/${pollId}/vote`, {
    method: 'POST',
    body: JSON.stringify({ option_id: optionId }),
  })
}

/**
 * Get aggregated live results snapshot from REST API
 * @param {string} pollId
 */
export async function getResults(pollId) {
  return request(`/polls/${pollId}/results`, {
    method: 'GET',
  })
}
