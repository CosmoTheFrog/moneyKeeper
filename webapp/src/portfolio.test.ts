import { describe, expect, it } from 'vitest'
import type { PortfolioItem } from './api'
import { bubbleSize, formatBP, packBubbles, shares } from './portfolio'

function item(value: number): PortfolioItem {
  return { currency: 'BTC', quantity_minor: 1, value_minor: value, cost_minor: 0, profit_minor: 0, profit_bp: null, realized_minor: 0, avg_price: '' }
}

describe('formatBP', () => {
  it.each([
    [1447, '+14,47%'],
    [-250, '−2,50%'],
    [0, '0,00%'],
    [5, '+0,05%'],
    [-5, '−0,05%'],
    [100000, '+1000,00%'],
  ])('%i → %s', (bp, want) => {
    expect(formatBP(bp)).toBe(want)
  })
})

describe('bubbleSize', () => {
  it('grows with the profit, symmetric for loss, capped', () => {
    expect(bubbleSize(0)).toBe(56)
    expect(bubbleSize(null)).toBe(56)
    expect(bubbleSize(1447)).toBeGreaterThan(bubbleSize(100))
    expect(bubbleSize(-1447)).toBe(bubbleSize(1447))
    expect(bubbleSize(10_000)).toBe(150)
    expect(bubbleSize(1_000_000)).toBe(150)
  })
})

describe('packBubbles', () => {
  const sizes = [150, 56, 80, 56, 100, 64, 56, 120]

  it('keeps every circle inside the field without overlaps', () => {
    const { placed, height } = packBubbles(sizes, 340)
    expect(placed).toHaveLength(sizes.length)
    placed.forEach((p, i) => {
      const r = (sizes[i] ?? 0) / 2
      expect(p.x - r).toBeGreaterThanOrEqual(0)
      expect(p.x + r).toBeLessThanOrEqual(340)
      expect(p.y - r).toBeGreaterThanOrEqual(0)
      expect(p.y + r).toBeLessThanOrEqual(height)
      placed.slice(i + 1).forEach((q, k) => {
        expect(Math.hypot(p.x - q.x, p.y - q.y)).toBeGreaterThanOrEqual(r + (sizes[i + 1 + k] ?? 0) / 2)
      })
    })
  })

  it('handles no circles and a single one', () => {
    expect(packBubbles([], 340).placed).toEqual([])
    expect(packBubbles([100], 340).placed).toEqual([{ x: 170, y: 50 }])
  })
})

describe('shares', () => {
  it('adds up to one hundred even when rounding drops pieces', () => {
    expect(shares([item(1), item(1), item(1)]).reduce((a, b) => a + b, 0)).toBe(100)
    expect(shares([item(50000), item(30000), item(20000)])).toEqual([50, 30, 20])
  })

  it('gives zero when nothing is worth anything', () => {
    expect(shares([item(0), item(0)])).toEqual([0, 0])
    expect(shares([])).toEqual([])
  })

  it('keeps small positions visible', () => {
    const s = shares([item(99_000), item(1_000)])
    expect(s).toEqual([99, 1])
  })
})
