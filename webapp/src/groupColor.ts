import { createContext } from 'react'
import type { Category, CategoryGroup } from './api'

// Оттенки групп идут по кругу из шести. Группа получает цвет по своему месту в списке и несёт его в значки,
// кольца и разбивки.
export function groupColors(groups: CategoryGroup[]): Map<number, string> {
  return new Map(groups.map((g, i) => [g.id, `var(--g${(i % 6) + 1})`]))
}

// categoryColors: цвет каждой категории это цвет её группы.
export function categoryColors(groups: CategoryGroup[], categories: Category[]): Map<number, string> {
  const byGroup = groupColors(groups)
  return new Map(categories.flatMap((c) => (byGroup.has(c.group_id) ? [[c.id, byGroup.get(c.group_id) as string] as const] : [])))
}

// Цвета категорий доступны всем спискам записей без передачи через каждый экран.
export const CategoryColorContext = createContext<Map<number, string>>(new Map())
