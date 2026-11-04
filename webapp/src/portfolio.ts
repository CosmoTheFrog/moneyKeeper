import type { PortfolioItem } from './api'

// formatBP показывает прибыль в сотых долях процента: 1447 → «+14,47%», -250 → «−2,50%».
export function formatBP(bp: number): string {
  const abs = Math.abs(bp)
  const sign = bp < 0 ? '−' : bp > 0 ? '+' : ''
  return `${sign}${Math.trunc(abs / 100)},${String(abs % 100).padStart(2, '0')}%`
}

export type Rect = { x: number; y: number; w: number; h: number }

// treemap раскладывает веса по прямоугольнику w×h плитками, близкими к квадратам (squarified). Площадь плитки
// пропорциональна весу, порядок результата совпадает с порядком weights. Нулевой и отрицательный вес даёт пустую плитку.
export function treemap(weights: number[], w: number, h: number): Rect[] {
  const out: Rect[] = weights.map(() => ({ x: 0, y: 0, w: 0, h: 0 }))
  const total = weights.reduce((sum, v) => sum + Math.max(v, 0), 0)
  if (total <= 0 || w <= 0 || h <= 0) return out

  const area = (i: number) => ((weights[i] ?? 0) * w * h) / total
  const order = weights
    .map((v, i) => i)
    .filter((i) => (weights[i] ?? 0) > 0)
    .sort((a, b) => (weights[b] ?? 0) - (weights[a] ?? 0) || a - b)

  // worst это худшее соотношение сторон в ряду, когда он ложится вдоль стороны side.
  const worst = (row: number[], side: number) => {
    const areas = row.map(area)
    const sum = areas.reduce((a, b) => a + b, 0)
    return Math.max((side * side * Math.max(...areas)) / (sum * sum), (sum * sum) / (side * side * Math.min(...areas)))
  }

  let x = 0
  let y = 0
  let rw = w
  let rh = h
  let row: number[] = []
  const flush = () => {
    const sum = row.reduce((a, i) => a + area(i), 0)
    if (rw >= rh) {
      // Столбец слева на всю высоту.
      const colW = sum / rh
      let top = y
      for (const i of row) {
        const tileH = area(i) / colW
        out[i] = { x, y: top, w: colW, h: tileH }
        top += tileH
      }
      x += colW
      rw -= colW
    } else {
      // Ряд сверху на всю ширину.
      const rowH = sum / rw
      let left = x
      for (const i of row) {
        const tileW = area(i) / rowH
        out[i] = { x: left, y, w: tileW, h: rowH }
        left += tileW
      }
      y += rowH
      rh -= rowH
    }
    row = []
  }
  for (const i of order) {
    const side = Math.min(rw, rh)
    if (row.length === 0 || worst([...row, i], side) <= worst(row, side)) {
      row.push(i)
    } else {
      flush()
      row.push(i)
    }
  }
  if (row.length > 0) flush()
  return out
}

export type HeatTone = { tone: 'up' | 'down' | 'flat'; strength: number }

// heatTone: цвет плитки по прибыли в сотых долях процента. Насыщенность растёт до 30% и дальше не меняется.
export function heatTone(bp: number | null): HeatTone {
  if (bp === null || bp === 0) return { tone: 'flat', strength: 0 }
  return { tone: bp > 0 ? 'up' : 'down', strength: Math.min(Math.abs(bp) / 3000, 1) }
}

// shares: доли позиций в стоимости портфеля в процентах, суммируются в сто. Позиции без стоимости получают ноль.
export function shares(items: PortfolioItem[]): number[] {
  const total = items.reduce((sum, i) => sum + i.value_minor, 0)
  if (total <= 0) return items.map(() => 0)
  const raw = items.map((i) => (i.value_minor * 100) / total)
  const floors = raw.map(Math.floor)
  // Остаток до ста раздаём позициям с самой большой дробной частью, чтобы сумма сошлась.
  let left = 100 - floors.reduce((a, b) => a + b, 0)
  const order = raw.map((r, i) => ({ i, frac: r - Math.floor(r) })).sort((a, b) => b.frac - a.frac)
  for (const { i } of order) {
    if (left <= 0) break
    floors[i] = (floors[i] ?? 0) + 1
    left--
  }
  return floors
}
