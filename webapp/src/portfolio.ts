import type { PortfolioItem } from './api'

// formatBP показывает прибыль в сотых долях процента: 1447 → «+14,47%», -250 → «−2,50%».
export function formatBP(bp: number): string {
  const abs = Math.abs(bp)
  const sign = bp < 0 ? '−' : bp > 0 ? '+' : ''
  return `${sign}${Math.trunc(abs / 100)},${String(abs % 100).padStart(2, '0')}%`
}

// bubbleSize: диаметр пузырька в пикселях по прибыли в процентах, от 56 до 150.
export function bubbleSize(bp: number | null): number {
  const percent = Math.min(Math.abs(bp ?? 0) / 100, 100)
  return Math.round(56 + percent * 0.94)
}

export type Placed = { x: number; y: number }

const STEP_DEG = 12

// packBubbles раскладывает круги разного диаметра в полосе шириной width плотной кучей, самый большой в середине сверху.
// Возвращает центры в порядке входных диаметров и высоту поля. Каждый следующий круг (от большего к меньшему) касается
// уже стоящего и выбирает свободное место ближе к середине кучи.
export function packBubbles(diameters: number[], width: number, gap = 4): { placed: Placed[]; height: number } {
  const order = diameters.map((d, i) => ({ d, i })).sort((a, b) => b.d - a.d || a.i - b.i)
  const placed: { x: number; y: number; r: number }[] = []
  const out: Placed[] = diameters.map(() => ({ x: 0, y: 0 }))

  for (const { d, i } of order) {
    const r = d / 2
    let best: { x: number; y: number } | null = null
    if (placed.length === 0) {
      best = { x: width / 2, y: r }
    } else {
      const cx = placed.reduce((s, p) => s + p.x, 0) / placed.length
      const cy = placed.reduce((s, p) => s + p.y, 0) / placed.length
      let bestScore = Infinity
      for (const p of placed) {
        for (let deg = 0; deg < 360; deg += STEP_DEG) {
          const a = (deg * Math.PI) / 180
          const x = p.x + Math.cos(a) * (p.r + r + gap)
          const y = p.y + Math.sin(a) * (p.r + r + gap)
          if (x - r < 0 || x + r > width || y - r < 0) continue
          if (placed.some((q) => Math.hypot(q.x - x, q.y - y) < q.r + r + gap - 0.01)) continue
          const score = Math.hypot(x - cx, y - cy)
          if (score < bestScore) {
            bestScore = score
            best = { x, y }
          }
        }
      }
    }
    // Узкое поле: если места рядом нет, круг ставится под всей кучей.
    const spot = best ?? { x: width / 2, y: Math.max(...placed.map((q) => q.y + q.r), 0) + r + gap }
    placed.push({ ...spot, r })
    out[i] = spot
  }
  return { placed: out, height: Math.max(...placed.map((q) => q.y + q.r), 0) + gap }
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
