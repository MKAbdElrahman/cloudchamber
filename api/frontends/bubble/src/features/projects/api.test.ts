import { describe, expect, it } from 'vitest'

import { projectErrorMessage } from './api'

describe('projectErrorMessage', () => {
  it('turns an invalid title into a title rule', () => {
    const message = `validate: ${JSON.stringify([{ field: 'name', error: 'invalid title "  "' }])}`
    const err = new Error(message)
    expect(projectErrorMessage(err)).toBe('Use a title of 1 to 80 characters.')
  })

  it('keeps a plain message', () => {
    expect(projectErrorMessage(new Error('Project name is required.'))).toBe('Project name is required.')
  })
})