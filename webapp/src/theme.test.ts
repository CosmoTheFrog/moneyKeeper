import { describe, expect, it } from 'vitest'
import { parseTheme } from './theme'

describe('parseTheme', () => {
  it.each([
    ['pop', 'pop'],
    ['bento', 'bento'],
    ['light', 'bento'],
    ['dark', 'bento'],
    ['auto', 'bento'],
    ['', 'bento'],
    [null, 'bento'],
  ])('%s → %s', (raw, want) => {
    expect(parseTheme(raw)).toBe(want)
  })
})
