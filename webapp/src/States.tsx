import type { ReactNode } from 'react'
import { t } from './i18n'

type SkeletonProps = { w?: string | number; h?: number; round?: boolean }

// Skeleton: серый контур, который мерцает, пока приходят данные. Место занято сразу, поэтому после загрузки ничего не прыгает.
export function Skeleton({ w = '100%', h = 14, round = false }: SkeletonProps) {
  return <span className={round ? 'sk sk-round' : 'sk'} style={{ width: w, height: h }} aria-hidden="true" />
}

function SkeletonRow() {
  return (
    <div className="sk-row">
      <Skeleton w={42} h={42} />
      <div className="sk-col">
        <Skeleton w="60%" h={13} />
        <Skeleton w="38%" h={10} />
      </div>
      <Skeleton w={56} h={14} />
    </div>
  )
}

// ScreenSkeleton: заготовка первого экрана, пока загружается приложение.
export function ScreenSkeleton() {
  return (
    <main className="screen" aria-busy="true" aria-label={t.loading}>
      <div className="sk-head">
        <Skeleton w={150} h={30} />
        <Skeleton w={38} h={38} round />
      </div>
      <div className="card sk-hero">
        <Skeleton w={90} h={11} />
        <Skeleton w={180} h={34} />
        <Skeleton h={52} />
      </div>
      <div className="sk-title">
        <Skeleton w={70} h={11} />
      </div>
      <div className="card list-card">
        <SkeletonRow />
        <SkeletonRow />
        <SkeletonRow />
      </div>
    </main>
  )
}

type EmptyProps = {
  title: string
  text: string
  action?: { label: string; onClick: () => void }
  children?: ReactNode
}

// EmptyState: пустой список с подсказкой и действием.
export function EmptyState({ title, text, action, children }: EmptyProps) {
  return (
    <div className="card state">
      <div className="state-art" aria-hidden="true">
        <svg viewBox="0 0 24 24">
          <path d="M12 3c-4 0-7 1.3-7 3v12c0 1.7 3 3 7 3s7-1.3 7-3V6c0-1.7-3-3-7-3zM5 12c0 1.7 3 3 7 3s7-1.3 7-3M5 6c0 1.7 3 3 7 3s7-1.3 7-3" />
        </svg>
      </div>
      <h3>{title}</h3>
      <p>{text}</p>
      {action !== undefined && (
        <button type="button" className="primary state-btn" onClick={action.onClick}>
          {action.label}
        </button>
      )}
      {children}
    </div>
  )
}

// ErrorState: не удалось загрузить; данные на месте, можно повторить.
export function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <main className="screen">
      <div className="card state state-bad">
        <div className="state-art" aria-hidden="true">
          <svg viewBox="0 0 24 24">
            <path d="M3 3l18 18M7 18h9.5a4 4 0 001.4-7.7A6 6 0 009 7M5.5 9.5A4.5 4.5 0 007 18" />
          </svg>
        </div>
        <h3>{t.states.errorTitle}</h3>
        <p>{t.states.errorText}</p>
        <button type="button" className="primary state-btn" onClick={onRetry}>
          {t.retry}
        </button>
      </div>
    </main>
  )
}
