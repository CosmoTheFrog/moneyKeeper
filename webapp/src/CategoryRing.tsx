import { categoryIcon } from './categoryIcons'
import type { RowStatus } from './budget'

type Props = {
  name: string
  // Какая часть доступного потрачена, 0..100.
  fill: number
  // Какая часть месяца прошла, 0..100; null, если месяц не текущий.
  pace: number | null
  status: RowStatus
}

const SIZE = 44
const R = 19
const C = 2 * Math.PI * R

// CategoryRing: значок категории в кольце расхода. Цвет кольца берётся из --gc (оттенок группы) или из статуса.
export function CategoryRing({ name, fill, pace, status }: Props) {
  const icon = categoryIcon(name)
  const mid = SIZE / 2
  let tick = null
  if (pace !== null) {
    const a = (pace / 100) * 2 * Math.PI - Math.PI / 2
    const at = (d: number) => [mid + (R + d) * Math.cos(a), mid + (R + d) * Math.sin(a)] as const
    const [x1, y1] = at(-4)
    const [x2, y2] = at(4)
    tick = <line className="ring-tick" x1={x1.toFixed(1)} y1={y1.toFixed(1)} x2={x2.toFixed(1)} y2={y2.toFixed(1)} />
  }
  return (
    <span className={`ring ring-${status}`} aria-hidden="true">
      <svg viewBox={`0 0 ${SIZE} ${SIZE}`}>
        <circle className="ring-track" cx={mid} cy={mid} r={R} />
        <circle
          className="ring-arc"
          cx={mid}
          cy={mid}
          r={R}
          style={{ '--c': C.toFixed(2) } as React.CSSProperties}
          strokeDasharray={C.toFixed(2)}
          strokeDashoffset={(C * (1 - Math.min(Math.max(fill, 0), 100) / 100)).toFixed(2)}
        />
        {tick}
      </svg>
      <span className="ring-icon">
        {icon !== null ? (
          <svg viewBox="0 0 24 24">
            <path d={icon} />
          </svg>
        ) : (
          <b>{[...name.trim()][0]?.toUpperCase() ?? ''}</b>
        )}
      </span>
    </span>
  )
}
