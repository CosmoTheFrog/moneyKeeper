import { useContext, useState, type CSSProperties, type ReactNode } from 'react'
import type { Account } from './api'
import { DisplayCurrency } from './DisplayCurrency'
import { ImportScreen } from './ImportScreen'
import { t } from './i18n'
import { PrivacyContext } from './MoneyText'
import { ThemeContext, ThemeSettings } from './ThemeSettings'

type Props = {
  name: string
  baseCurrency: string
  accounts: Account[]
  currencies: string[]
  display: string
  onDisplay: (code: string) => void
  // Записи изменились (импорт): данные приложения нужно перечитать.
  onChanged: () => void
  onBack: () => void
}

const ICONS = {
  palette: 'M12 3a9 9 0 100 18c1.1 0 2-.9 2-2 0-.5-.2-1-.5-1.3-.3-.3-.5-.8-.5-1.2 0-1.1.9-2 2-2h2.4A3.6 3.6 0 0021 11c0-4.4-4-8-9-8zM7.5 11h.01M10 7.5h.01M14.5 7.5h.01',
  globe: 'M12 3a9 9 0 100 18 9 9 0 000-18zM3 12h18M12 3c2.5 2.7 3.8 5.7 3.8 9S14.5 18.3 12 21c-2.5-2.7-3.8-5.7-3.8-9S9.5 5.7 12 3z',
  eye: 'M3 3l18 18M10.6 6.1A9.6 9.6 0 0112 6c6 0 10 6 10 6a17 17 0 01-3.2 3.7M6.4 6.9A17 17 0 002 12s4 6 10 6a9.8 9.8 0 004-.9',
  upload: 'M12 15V3M7 8l5-5 5 5M5 21h14',
}

function Row({ icon, tint, title, sub, end, onClick }: { icon: keyof typeof ICONS; tint: string; title: string; sub?: string; end?: ReactNode; onClick?: () => void }) {
  const body = (
    <>
      <span className="avatar" style={{ '--gc': tint } as CSSProperties} aria-hidden="true">
        <svg viewBox="0 0 24 24">
          <path d={ICONS[icon]} />
        </svg>
      </span>
      <span className="more-mid">
        <span className="more-title">{title}</span>
        {sub !== undefined && <span className="more-sub">{sub}</span>}
      </span>
      {end}
    </>
  )
  return onClick !== undefined ? (
    <button type="button" className="more-row" onClick={onClick}>
      {body}
      <span className="chev" aria-hidden="true">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round">
          <path d="M9 6l6 6-6 6" />
        </svg>
      </span>
    </button>
  ) : (
    <div className="more-row">{body}</div>
  )
}

// Ещё: настройки и работа с данными в одном месте.
export function MoreScreen({ name, baseCurrency, accounts, currencies, display, onDisplay, onChanged, onBack }: Props) {
  const { mode } = useContext(ThemeContext)
  const { hidden, toggle } = useContext(PrivacyContext)
  const [theme, setTheme] = useState(false)
  const [importing, setImporting] = useState(false)

  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onBack}>
          {t.back}
        </button>
      </header>
      <h1>{t.more.title}</h1>

      <div className="card more-profile">
        <span className="more-ph" aria-hidden="true">
          {[...name.trim()][0]?.toUpperCase() ?? ''}
        </span>
        <div>
          <b>{name}</b>
          <span>
            {t.personalBudget} · {baseCurrency}
          </span>
        </div>
      </div>

      <h2 className="day">{t.more.view}</h2>
      <div className="card list-card">
        <Row icon="palette" tint="var(--g3)" title={t.theme.open} sub={t.theme.modes[mode]} onClick={() => setTheme(true)} />
        <Row icon="globe" tint="var(--g4)" title={t.more.displayCurrency} sub={t.rates.countInHint} end={<DisplayCurrency display={display} currencies={currencies} onChange={onDisplay} />} />
        <label className="more-row">
          <span className="avatar" style={{ '--gc': 'var(--g5)' } as CSSProperties} aria-hidden="true">
            <svg viewBox="0 0 24 24">
              <path d={ICONS.eye} />
            </svg>
          </span>
          <span className="more-mid">
            <span className="more-title">{t.more.hideSums}</span>
            <span className="more-sub">{t.more.hideSumsHint}</span>
          </span>
          <span className="switch">
            <input type="checkbox" checked={hidden} onChange={toggle} />
            <span className="switch-track" />
          </span>
        </label>
      </div>

      <h2 className="day">{t.more.data}</h2>
      <div className="card list-card">
        <Row icon="upload" tint="var(--g2)" title={t.accounts.import} sub={t.more.importHint} onClick={() => setImporting(true)} />
      </div>

      {theme && <ThemeSettings onClose={() => setTheme(false)} />}
      {importing && (
        <ImportScreen
          accounts={accounts}
          onClose={() => setImporting(false)}
          onDone={() => {
            setImporting(false)
            onChanged()
          }}
        />
      )}
    </main>
  )
}
