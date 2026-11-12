import { createContext } from 'react'
import type { Category } from './api'

// Оттенки идут по кругу из шести: категория получает цвет по своему месту в списке и несёт его в значки,
// кольца и разбивки.
export function categoryColors(categories: Category[]): Map<number, string> {
  return new Map(categories.map((c, i) => [c.id, `var(--g${(i % 6) + 1})`]))
}

// Цвета категорий доступны всем спискам записей без передачи через каждый экран.
export const CategoryColorContext = createContext<Map<number, string>>(new Map())
