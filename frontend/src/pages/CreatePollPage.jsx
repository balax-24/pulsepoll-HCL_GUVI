import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createPoll } from '../api/polls.js'
import { Alert } from '../components/Alert.jsx'
import { IconCheckCircle, IconCopy, IconCheck, IconPlus, IconX, IconArrowRight, IconExternalLink } from '../components/Icons.jsx'

export function CreatePollPage() {
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [expiresAt, setExpiresAt] = useState('')
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [createdPoll, setCreatedPoll] = useState(null)
  const [copied, setCopied] = useState(false)

  const navigate = useNavigate()

  const handleAddOption = () => {
    if (options.length >= 10) return
    setOptions((prev) => [...prev, ''])
  }

  const handleRemoveOption = (index) => {
    if (options.length <= 2) return
    setOptions((prev) => prev.filter((_, i) => i !== index))
  }

  const handleOptionChange = (index, value) => {
    setOptions((prev) => {
      const updated = [...prev]
      updated[index] = value
      return updated
    })
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setFormError('')

    const trimmedQuestion = question.trim()
    if (trimmedQuestion.length < 3 || trimmedQuestion.length > 255) {
      setFormError('Question must be between 3 and 255 characters.')
      return
    }

    if (options.length < 2 || options.length > 10) {
      setFormError('A poll must have between 2 and 10 options.')
      return
    }

    const cleanedOptions = []
    const seen = new Set()

    for (let i = 0; i < options.length; i++) {
      const opt = options[i].trim()
      if (!opt) {
        setFormError(`Option ${i + 1} cannot be empty.`)
        return
      }
      if (opt.length > 150) {
        setFormError(`Option ${i + 1} cannot exceed 150 characters.`)
        return
      }
      const lower = opt.toLowerCase()
      if (seen.has(lower)) {
        setFormError(`Duplicate option detected: "${opt}". Each option must be distinct.`)
        return
      }
      seen.add(lower)
      cleanedOptions.push(opt)
    }

    let formattedExpiresAt = undefined
    if (expiresAt) {
      const expDate = new Date(expiresAt)
      if (isNaN(expDate.getTime()) || expDate <= new Date()) {
        setFormError('Expiration deadline must be a valid future date and time.')
        return
      }
      formattedExpiresAt = expDate.toISOString()
    }

    setIsSubmitting(true)
    try {
      const payload = {
        question: trimmedQuestion,
        options: cleanedOptions,
      }
      if (formattedExpiresAt) {
        payload.expires_at = formattedExpiresAt
      }

      const response = await createPoll(payload)
      setCreatedPoll(response.poll || response)
    } catch (err) {
      setFormError(err.message || 'Failed to create poll. Please check details and try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  const shareUrl = createdPoll ? `${window.location.origin}/polls/${createdPoll.id}` : ''

  const handleCopyLink = async () => {
    if (!shareUrl) return
    try {
      await navigator.clipboard.writeText(shareUrl)
      setCopied(true)
      setTimeout(() => setCopied(false), 2500)
    } catch {
      setCopied(false)
    }
  }

  const handleReset = () => {
    setCreatedPoll(null)
    setQuestion('')
    setOptions(['', ''])
    setExpiresAt('')
    setFormError('')
    setCopied(false)
  }

  return (
    <div className="create-poll-container">
      {createdPoll ? (
        /* Success Presentation State */
        <div className="poll-created-success-card">
          <div className="success-icon-badge">
            <IconCheckCircle size={32} className="text-emerald" />
          </div>
          <h1 className="success-title">Poll Created Successfully!</h1>
          <p className="success-subtitle">
            Your live poll is ready and accepting real-time responses.
          </p>

          <div className="created-poll-summary">
            <div className="summary-question-label">Active Question</div>
            <h2 className="summary-question">"{createdPoll.question}"</h2>
            <div className="summary-options-list">
              {createdPoll.options?.map((opt, i) => (
                <span key={opt.id || i} className="summary-option-chip">
                  <span className="chip-index">{String(i + 1).padStart(2, '0')}</span>
                  <span>{opt.text}</span>
                </span>
              ))}
            </div>
          </div>

          <div className="share-link-box">
            <label className="share-label" htmlFor="poll-share-link">
              Shareable Audience Link
            </label>
            <div className="share-input-row">
              <input
                id="poll-share-link"
                type="text"
                readOnly
                value={shareUrl}
                className="form-input share-url-input"
              />
              <button
                type="button"
                onClick={handleCopyLink}
                className="btn btn-secondary copy-action-btn"
                aria-label="Copy Link"
              >
                {copied ? (
                  <>
                    <IconCheck size={16} className="text-emerald" />
                    <span>✓ Copied!</span>
                  </>
                ) : (
                  <>
                    <IconCopy size={16} />
                    <span>Copy Link</span>
                  </>
                )}
              </button>
            </div>
            <p className="share-hint">
              Anyone with this link can vote anonymously and observe live results without signing up.
            </p>
          </div>

          <div className="success-action-buttons">
            <button
              type="button"
              onClick={() => navigate(`/polls/${createdPoll.id}`)}
              className="btn btn-primary"
            >
              <span>Open Live Poll</span>
              <IconArrowRight size={16} />
            </button>
            <button
              type="button"
              onClick={() => navigate(`/polls/${createdPoll.id}/manage`)}
              className="btn btn-outline"
            >
              Manage Poll
            </button>
            <button
              type="button"
              onClick={handleReset}
              className="btn btn-ghost"
            >
              + Create Another Poll
            </button>
          </div>
        </div>
      ) : (
        /* Poll Creation Form */
        <div className="create-poll-card">
          <div className="create-poll-header">
            <div className="header-meta-tag">New Poll</div>
            <h1 className="page-heading">Create a Live Poll</h1>
            <p className="page-subheading">
              Configure your question and answer choices. Connected audiences will view real-time updates.
            </p>
          </div>

          {formError && (
            <Alert
              type="error"
              message={formError}
              onClose={() => setFormError('')}
              className="mb-4"
            />
          )}

          <form onSubmit={handleSubmit} className="poll-form" noValidate>
            {/* Question Field */}
            <div className="form-group">
              <label htmlFor="poll-question" className="form-label">
                Poll Question <span className="required-mark">*</span>
              </label>
              <input
                id="poll-question"
                type="text"
                className="form-input form-input-lg"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="e.g. Which cloud architecture fits our roadmap best?"
                maxLength={255}
                required
                disabled={isSubmitting}
              />
              <div className="input-meta-row">
                <span className="input-hint">3 to 255 characters</span>
                <span className="char-count">{question.length} / 255</span>
              </div>
            </div>

            {/* Dynamic Options List */}
            <div className="form-group options-group">
              <div className="label-with-badge">
                <label className="form-label">
                  Voting Options <span className="required-mark">*</span>
                </label>
                <span className="count-pill">
                  {options.length} of 10 Options
                </span>
              </div>

              <div className="options-input-list">
                {options.map((opt, index) => (
                  <div key={index} className="option-row">
                    <span className="option-index-pill">
                      {String(index + 1).padStart(2, '0')}
                    </span>
                    <input
                      type="text"
                      className="form-input option-input"
                      value={opt}
                      onChange={(e) => handleOptionChange(index, e.target.value)}
                      placeholder={`Option ${index + 1}`}
                      maxLength={150}
                      required
                      disabled={isSubmitting}
                      aria-label={`Option ${index + 1}`}
                    />
                    {options.length > 2 && (
                      <button
                        type="button"
                        onClick={() => handleRemoveOption(index)}
                        className="btn-remove-option"
                        title="Remove this option"
                        disabled={isSubmitting}
                        aria-label={`Remove option ${index + 1}`}
                      >
                        <IconX size={14} />
                      </button>
                    )}
                  </div>
                ))}
              </div>

              {options.length < 10 && (
                <button
                  type="button"
                  onClick={handleAddOption}
                  className="btn btn-sm btn-outline add-option-btn"
                  disabled={isSubmitting}
                >
                  <IconPlus size={14} />
                  <span>+ Add Option</span>
                </button>
              )}
            </div>

            {/* Optional Expiration Field */}
            <div className="form-group expiration-group">
              <label htmlFor="poll-expires" className="form-label">
                Expiration Deadline <span className="text-muted font-normal">(Optional)</span>
              </label>
              <input
                id="poll-expires"
                type="datetime-local"
                className="form-input"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
                disabled={isSubmitting}
              />
              <span className="input-hint">
                Leave empty if you wish to keep the poll open until manually closed.
              </span>
            </div>

            {/* Form Actions */}
            <div className="form-actions-row">
              <button
                type="button"
                onClick={() => navigate('/dashboard')}
                className="btn btn-ghost"
                disabled={isSubmitting}
              >
                Cancel
              </button>
              <button
                type="submit"
                className="btn btn-primary btn-submit-poll"
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <span className="btn-spinner-content">
                    <span className="spinner-inline" />
                    Publishing Poll...
                  </span>
                ) : (
                  'Publish Poll'
                )}
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  )
}
