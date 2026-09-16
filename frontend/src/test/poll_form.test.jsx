import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { CreatePollPage } from '../pages/CreatePollPage.jsx'
import * as pollsApi from '../api/polls.js'

vi.mock('../api/polls.js', () => ({
  createPoll: vi.fn(),
}))

describe('CreatePollPage Validation & Interactions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('rejects questions that are too short', async () => {
    render(
      <MemoryRouter>
        <CreatePollPage />
      </MemoryRouter>
    )

    const questionInput = screen.getByLabelText(/Poll Question/i)
    fireEvent.change(questionInput, { target: { value: 'Hi' } })

    const submitBtn = screen.getByRole('button', { name: /Publish Poll/i })
    fireEvent.click(submitBtn)

    expect(screen.getByText(/Question must be between 3 and 255 characters/i)).toBeInTheDocument()
    expect(pollsApi.createPoll).not.toHaveBeenCalled()
  })

  it('allows adding options up to 10 and removing down to 2', () => {
    render(
      <MemoryRouter>
        <CreatePollPage />
      </MemoryRouter>
    )

    // Starts with 2 options
    expect(screen.getAllByPlaceholderText(/Option /i)).toHaveLength(2)

    // Add 3rd option
    const addBtn = screen.getByRole('button', { name: /\+ Add Option/i })
    fireEvent.click(addBtn)
    expect(screen.getAllByPlaceholderText(/Option /i)).toHaveLength(3)

    // Remove 3rd option
    const removeBtns = screen.getAllByTitle(/Remove this option/i)
    expect(removeBtns.length).toBeGreaterThan(0)
    fireEvent.click(removeBtns[removeBtns.length - 1])
    expect(screen.getAllByPlaceholderText(/Option /i)).toHaveLength(2)
  })

  it('detects duplicate options case-insensitively', async () => {
    render(
      <MemoryRouter>
        <CreatePollPage />
      </MemoryRouter>
    )

    const questionInput = screen.getByLabelText(/Poll Question/i)
    fireEvent.change(questionInput, { target: { value: 'Which language do you prefer?' } })

    const optionInputs = screen.getAllByPlaceholderText(/Option /i)
    fireEvent.change(optionInputs[0], { target: { value: 'TypeScript' } })
    fireEvent.change(optionInputs[1], { target: { value: 'typescript' } })

    const submitBtn = screen.getByRole('button', { name: /Publish Poll/i })
    fireEvent.click(submitBtn)

    expect(screen.getByText(/Duplicate option detected/i)).toBeInTheDocument()
    expect(pollsApi.createPoll).not.toHaveBeenCalled()
  })

  it('submits valid poll and shows success state with copy link', async () => {
    pollsApi.createPoll.mockResolvedValue({
      id: 'poll-123',
      question: 'Best database?',
      options: [
        { id: 'opt-1', text: 'PostgreSQL' },
        { id: 'opt-2', text: 'MongoDB' },
      ],
      status: 'active',
    })

    render(
      <MemoryRouter>
        <CreatePollPage />
      </MemoryRouter>
    )

    const questionInput = screen.getByLabelText(/Poll Question/i)
    fireEvent.change(questionInput, { target: { value: 'Best database?' } })

    const optionInputs = screen.getAllByPlaceholderText(/Option /i)
    fireEvent.change(optionInputs[0], { target: { value: 'PostgreSQL' } })
    fireEvent.change(optionInputs[1], { target: { value: 'MongoDB' } })

    const submitBtn = screen.getByRole('button', { name: /Publish Poll/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText(/Poll Created Successfully!/i)).toBeInTheDocument()
      expect(screen.getByText(/Open Live Poll/i)).toBeInTheDocument()
      expect(screen.getByText(/Copy Link/i)).toBeInTheDocument()
    })
  })
})
