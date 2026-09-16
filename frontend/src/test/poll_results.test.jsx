import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PollResults } from '../components/PollResults.jsx'

describe('PollResults Component', () => {
  it('renders all options with vote counts and percentages', () => {
    const mockOptions = [
      { option_id: 'opt-1', text: 'Go / Gin', count: 12, percentage: 60 },
      { option_id: 'opt-2', text: 'Node / Express', count: 8, percentage: 40 },
    ]

    render(<PollResults options={mockOptions} totalVotes={20} />)

    expect(screen.getByText('Live Results')).toBeInTheDocument()
    expect(screen.getByText('20')).toBeInTheDocument()
    expect(screen.getByText('Go / Gin')).toBeInTheDocument()
    expect(screen.getByText('Node / Express')).toBeInTheDocument()
    expect(screen.getByText('60.0%')).toBeInTheDocument()
    expect(screen.getByText('40.0%')).toBeInTheDocument()
  })

  it('highlights leading option and user voted option', () => {
    const mockOptions = [
      { option_id: 'opt-1', text: 'Python', count: 15, percentage: 75 },
      { option_id: 'opt-2', text: 'Ruby', count: 5, percentage: 25 },
    ]

    render(
      <PollResults
        options={mockOptions}
        totalVotes={20}
        userVotedOptionId="opt-2"
        isClosed={false}
      />
    )

    // Option 1 has leading badge
    expect(screen.getByText('Leading')).toBeInTheDocument()

    // Option 2 has user vote badge
    expect(screen.getByText('Your Vote')).toBeInTheDocument()
  })

  it('displays closed badge when isClosed is true', () => {
    const mockOptions = [
      { option_id: 'opt-1', text: 'Option A', count: 1, percentage: 100 },
    ]

    render(<PollResults options={mockOptions} totalVotes={1} isClosed={true} />)

    expect(screen.getByText('Voting Closed')).toBeInTheDocument()
  })

  it('handles zero total votes safely without NaN errors', () => {
    const mockOptions = [
      { option_id: 'opt-1', text: 'Option A', count: 0, percentage: 0 },
      { option_id: 'opt-2', text: 'Option B', count: 0, percentage: 0 },
    ]

    render(<PollResults options={mockOptions} totalVotes={0} />)

    expect(screen.getByText('0')).toBeInTheDocument()
    expect(screen.getAllByText('0.0%')).toHaveLength(2)
  })
})
