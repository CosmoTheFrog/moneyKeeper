import { parseAmount } from './money'

export type Tone = 'open' | 'done' | 'over'

// toBudgetTone: больше нуля осталось распределить, ноль всё распределено, меньше нуля заложено больше, чем есть.
export function toBudgetTone(toBudgetMinor: number): Tone {
  if (toBudgetMinor > 0) return 'open'
  return toBudgetMinor === 0 ? 'done' : 'over'
}

// fillPercent: какая часть заложенного потрачена, для полоски. Потрачено сверх заложенного это целая полоска;
// потрачено без заложенного тоже.
export function fillPercent(spentMinor: number, budgetedMinor: number): number {
  if (spentMinor <= 0) return 0
  if (budgetedMinor <= 0) return 100
  return Math.min(100, Math.round((spentMinor * 100) / budgetedMinor))
}

// parseBudgeted разбирает сумму из поля: пусто и ноль значат «ничего не закладывать». Мусор и лишние знаки дают null.
export function parseBudgeted(input: string, currency: string): number | null {
  const text = input.trim()
  if (text === '' || /^0+([.,]0*)?$/.test(text)) return 0
  return parseAmount(text, currency)
}
