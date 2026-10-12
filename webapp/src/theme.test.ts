import { describe, expect, it } from 'vitest'
import { parseMode, resolveTheme } from './theme'

describe('parseMode', () => {
  it.each([
    ['light', 'light'],
    ['dark', 'dark'],
    ['auto', 'auto'],
    ['blue', 'auto'],
    ['', 'auto'],
    [null, 'auto'],
  ])('%s → %s', (raw, want) => {
    expect(parseMode(raw)).toBe(want)
  })
})

describe('resolveTheme', () => {
  it('explicit choice wins over Telegram and system', () => {
    expect(resolveTheme('light', 'dark', true)).toBe('light')
    expect(resolveTheme('dark', 'light', false)).toBe('dark')
  })

  it('auto follows Telegram first', () => {
    expect(resolveTheme('auto', 'dark', false)).toBe('dark')
    expect(resolveTheme('auto', 'light', true)).toBe('light')
  })

  it('auto falls back to the system theme without Telegram', () => {
    expect(resolveTheme('auto', undefined, true)).toBe('dark')
    expect(resolveTheme('auto', undefined, false)).toBe('light')
    expect(resolveTheme('auto', 'weird', true)).toBe('dark')
  })
})
