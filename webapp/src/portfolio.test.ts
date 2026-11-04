import { describe, expect, it } from 'vitest'
import type { PortfolioItem } from './api'
import { formatBP, heatTone, shares, treemap, type Rect } from './portfolio'

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

describe('treemap', () => {
  const weights = [8400, 4100, 2650, 1900, 1370]
  const W = 340
  const H = 300
  const rects = treemap(weights, W, H)
  const total = weights.reduce((a, b) => a + b, 0)
  const area = (r: Rect) => r.w * r.h

  it('площадь плитки пропорциональна весу, вместе они закрывают прямоугольник', () => {
    expect(rects.reduce((sum, r) => sum + area(r), 0)).toBeCloseTo(W * H, 4)
    rects.forEach((r, i) => expect(area(r) / (W * H)).toBeCloseTo((weights[i] ?? 0) / total, 6))
  })

  it('плитки лежат внутри прямоугольника', () => {
    for (const r of rects) {
      expect(r.x).toBeGreaterThanOrEqual(-1e-9)
      expect(r.y).toBeGreaterThanOrEqual(-1e-9)
      expect(r.x + r.w).toBeLessThanOrEqual(W + 1e-9)
      expect(r.y + r.h).toBeLessThanOrEqual(H + 1e-9)
    }
  })

  it('плитки не пересекаются', () => {
    for (let i = 0; i < rects.length; i++) {
      for (let j = i + 1; j < rects.length; j++) {
        const a = rects[i]!
        const b = rects[j]!
        const dx = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x)
        const dy = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y)
        expect(dx > 1e-6 && dy > 1e-6).toBe(false)
      }
    }
  })

  it.each([
    ['один вес занимает всё', [5], [{ x: 0, y: 0, w: W, h: H }]],
    ['пустой список', [], []],
  ])('%s', (_name, ws, want) => {
    expect(treemap(ws, W, H)).toEqual(want)
  })

  it('нулевой вес даёт пустую плитку и не сбивает порядок', () => {
    const r = treemap([5, 0, 5], 100, 100)
    expect(area(r[1]!)).toBe(0)
    expect(area(r[0]!)).toBeCloseTo(5000, 6)
    expect(area(r[2]!)).toBeCloseTo(5000, 6)
  })

  it('нулевой размер поля даёт пустые плитки', () => {
    expect(treemap([1, 2], 0, 100).every((r) => area(r) === 0)).toBe(true)
  })
})

describe('heatTone', () => {
  it.each([
    [null, 'flat', 0],
    [0, 'flat', 0],
    [1500, 'up', 0.5],
    [-1500, 'down', 0.5],
    [3000, 'up', 1],
    [9000, 'up', 1],
    [-9000, 'down', 1],
  ])('%s → %s, насыщенность %s', (bp, tone, strength) => {
    expect(heatTone(bp)).toEqual({ tone, strength })
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
