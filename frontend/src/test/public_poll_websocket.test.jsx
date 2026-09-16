import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { PublicPollPage } from '../pages/PublicPollPage.jsx'
import * as pollsApi from '../api/polls.js'
import * as votesApi from '../api/votes.js'
import * as wsHook from '../hooks/usePollWebSocket.js'

vi.mock('../api/polls.js', () => ({
  getPublicPoll: vi.fn(),
}))

vi.mock('../api/votes.js', () => ({
  getResults: vi.fn(),
  castVote: vi.fn(),
}))

vi.mock('../hooks/usePollWebSocket.js', () => ({
  usePollWebSocket: vi.fn(),
}))

describe('PublicPollPage & Realtime WebSocket Events', () => {
  let wsCallbacks = {}

  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    wsCallbacks = {}

    // Capture the hook callbacks so tests can fire events directly
    wsHook.usePollWebSocket.mockImplementation((pollId, callbacks) => {
      wsCallbacks = callbacks
      return {
        status: 'connected',
        reconnectAttempts: 0,
        reconnect: vi.fn(),
      }
    })
  })

  it('renders poll and allows casting a vote', async () => {
    pollsApi.getPublicPoll.mockResolvedValue({
      poll: {
        id: 'poll-456',
        question: 'What is your preferred editor?',
        options: [
          { id: 'opt-a', text: 'Neovim' },
          { id: 'opt-b', text: 'VS Code' },
        ],
        status: 'active',
      },
    })

    votesApi.getResults.mockResolvedValue({
      poll_id: 'poll-456',
      total_votes: 0,
      results: [
        { option_id: 'opt-a', text: 'Neovim', votes: 0, percentage: 0 },
        { option_id: 'opt-b', text: 'VS Code', votes: 0, percentage: 0 },
      ],
    })

    votesApi.castVote.mockResolvedValue({
      poll_id: 'poll-456',
      option_id: 'opt-a',
      message: 'Vote recorded',
    })

    render(
      <MemoryRouter initialEntries={['/polls/poll-456']}>
        <Routes>
          <Route path="/polls/:id" element={<PublicPollPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('What is your preferred editor?')).toBeInTheDocument()
    })

    // Select Option A
    const radioA = screen.getByLabelText('Neovim')
    fireEvent.click(radioA)

    // Submit Vote
    const submitBtn = screen.getByRole('button', { name: /Submit Vote/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(votesApi.castVote).toHaveBeenCalledWith('poll-456', 'opt-a')
      expect(screen.getByText(/Your vote has been recorded!/i)).toBeInTheDocument()
    })
  })

  it('handles duplicate vote (409 / DUPLICATE_VOTE) gracefully', async () => {
    pollsApi.getPublicPoll.mockResolvedValue({
      poll: {
        id: 'poll-456',
        question: 'Duplicate vote test question',
        options: [
          { id: 'opt-1', text: 'Yes' },
          { id: 'opt-2', text: 'No' },
        ],
        status: 'active',
      },
    })

    votesApi.getResults.mockResolvedValue({
      poll_id: 'poll-456',
      total_votes: 1,
      results: [
        { option_id: 'opt-1', text: 'Yes', votes: 1, percentage: 100 },
        { option_id: 'opt-2', text: 'No', votes: 0, percentage: 0 },
      ],
    })

    const duplicateErr = new Error("You have already voted in this poll")
    duplicateErr.status = 409
    duplicateErr.code = 'DUPLICATE_VOTE'
    votesApi.castVote.mockRejectedValue(duplicateErr)

    render(
      <MemoryRouter initialEntries={['/polls/poll-456']}>
        <Routes>
          <Route path="/polls/:id" element={<PublicPollPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('Duplicate vote test question')).toBeInTheDocument()
    })

    const radio = screen.getByLabelText('Yes')
    fireEvent.click(radio)

    const submitBtn = screen.getByRole('button', { name: /Submit Vote/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText("You've already voted in this poll.")).toBeInTheDocument()
    })
  })

  it('updates live results upon receiving WebSocket vote_update event without refresh', async () => {
    pollsApi.getPublicPoll.mockResolvedValue({
      poll: {
        id: 'poll-789',
        question: 'Favorite Cloud Provider?',
        options: [
          { id: 'opt-aws', text: 'AWS' },
          { id: 'opt-gcp', text: 'GCP' },
        ],
        status: 'active',
      },
    })

    votesApi.getResults.mockResolvedValue({
      poll_id: 'poll-789',
      total_votes: 10,
      results: [
        { option_id: 'opt-aws', text: 'AWS', votes: 6, percentage: 60 },
        { option_id: 'opt-gcp', text: 'GCP', votes: 4, percentage: 40 },
      ],
    })

    render(
      <MemoryRouter initialEntries={['/polls/poll-789']}>
        <Routes>
          <Route path="/polls/:id" element={<PublicPollPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('Favorite Cloud Provider?')).toBeInTheDocument()
      expect(screen.getByText('60.0%')).toBeInTheDocument()
    })

    // Simulate incoming WebSocket vote_update for GCP
    act(() => {
      wsCallbacks.onVoteUpdate({
        type: 'vote_update',
        poll_id: 'poll-789',
        option_id: 'opt-gcp',
        count: 5,
        total_votes: 11,
      })
    })

    // Verify GCP updated to 5 votes and total to 11 votes
    await waitFor(() => {
      expect(screen.getByText('11')).toBeInTheDocument()
      expect(screen.getByText('5 votes')).toBeInTheDocument()
    })
  })

  it('synchronizes full state upon receiving results_snapshot WebSocket event', async () => {
    pollsApi.getPublicPoll.mockResolvedValue({
      poll: {
        id: 'poll-789',
        question: 'Favorite Cloud Provider?',
        options: [
          { id: 'opt-aws', text: 'AWS' },
          { id: 'opt-gcp', text: 'GCP' },
        ],
        status: 'active',
      },
    })

    votesApi.getResults.mockResolvedValue({
      poll_id: 'poll-789',
      total_votes: 0,
      results: [],
    })

    render(
      <MemoryRouter initialEntries={['/polls/poll-789']}>
        <Routes>
          <Route path="/polls/:id" element={<PublicPollPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('Favorite Cloud Provider?')).toBeInTheDocument()
    })

    // Simulate incoming WebSocket results_snapshot
    act(() => {
      wsCallbacks.onSnapshot({
        type: 'results_snapshot',
        poll_id: 'poll-789',
        options: [
          { option_id: 'opt-aws', text: 'AWS', count: 15, percentage: 75 },
          { option_id: 'opt-gcp', text: 'GCP', count: 5, percentage: 25 },
        ],
        total_votes: 20,
      })
    })

    await waitFor(() => {
      expect(screen.getByText('20')).toBeInTheDocument()
      expect(screen.getByText('75.0%')).toBeInTheDocument()
      expect(screen.getByText('25.0%')).toBeInTheDocument()
    })
  })

  it('disables voting and displays closed state upon receiving poll_closed event', async () => {
    pollsApi.getPublicPoll.mockResolvedValue({
      poll: {
        id: 'poll-789',
        question: 'Favorite Cloud Provider?',
        options: [
          { id: 'opt-aws', text: 'AWS' },
          { id: 'opt-gcp', text: 'GCP' },
        ],
        status: 'active',
      },
    })

    votesApi.getResults.mockResolvedValue({
      poll_id: 'poll-789',
      total_votes: 1,
      results: [
        { option_id: 'opt-aws', text: 'AWS', votes: 1, percentage: 100 },
        { option_id: 'opt-gcp', text: 'GCP', votes: 0, percentage: 0 },
      ],
    })

    render(
      <MemoryRouter initialEntries={['/polls/poll-789']}>
        <Routes>
          <Route path="/polls/:id" element={<PublicPollPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Submit Vote/i })).toBeInTheDocument()
    })

    // Fire WebSocket poll_closed event
    act(() => {
      wsCallbacks.onPollClosed({
        type: 'poll_closed',
        poll_id: 'poll-789',
      })
    })

    await waitFor(() => {
      // Vote button removed
      expect(screen.queryByRole('button', { name: /Submit Vote/i })).not.toBeInTheDocument()
      // Voting closed notice shown
      expect(screen.getByText(/Voting is now closed for this poll/i)).toBeInTheDocument()
    })
  })
})
