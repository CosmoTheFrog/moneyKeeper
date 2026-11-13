import { useContext, useState, type CSSProperties, type ReactNode } from 'react'
import { setTourSeen, type Account, type InterfaceMode } from './api'
import { DisplayCurrency } from './DisplayCurrency'
import { ExportScreen } from './ExportScreen'
import { ImportScreen } from './ImportScreen'
import { Modal } from './Modal'
import { ModePicker } from './ModeScreen'
import { setInterfaceMode } from './api'
import { t } from './i18n'
import { PrivacyContext } from './MoneyText'
import { ReminderSettings } from './ReminderSettings'
import { MyCurrencies } from './MyCurrencies'
import { ThemeContext, ThemeSettings } from './ThemeSettings'

type Props = {
  name: string
  baseCurrency: string
  accounts: Account[]
  interfaceMode: InterfaceMode | null
  currencies: string[]
  // Все валюты реестра, включённые пользователем (null, пока не выбирал) и те, что отключить нельзя.
  registry: string[]
  enabled: string[] | null
  pinned: string[]
  onEnabled: (list: string[]) => void
  coinsInRates: boolean
  onCoinsInRates: (on: boolean) => void
  display: string
  // Час ежедневного напоминания внести траты, null если выключено.
  reminderHour: number | null
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
  download: 'M12 3v12M7 10l5 5 5-5M5 21h14',
  bell: 'M18 8a6 6 0 10-12 0c0 7-3 9-3 9h18s-3-2-3-9zM13.7 21a2 2 0 01-3.4 0',
  layout: 'M4 5h16v14H4zM4 10h16M10 10v9',
  help: 'M12 21a9 9 0 100-18 9 9 0 000 18zM9.5 9.5a2.5 2.5 0 114 2c-.9.6-1.5 1.1-1.5 2M12 17h.01',
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
export function MoreScreen({ name, baseCurrency, accounts, interfaceMode, currencies, registry, enabled, pinned, onEnabled, coinsInRates, onCoinsInRates, display, reminderHour, onDisplay, onChanged, onBack }: Props) {
  const { theme: appearance } = useContext(ThemeContext)
  const { hidden, toggle } = useContext(PrivacyContext)
  const [theme, setTheme] = useState(false)
  const [importing, setImporting] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [reminding, setReminding] = useState(false)
  const [switching, setSwitching] = useState(false)
  const [choosing, setChoosing] = useState(false)

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
        <Row icon="layout" tint="var(--g1)" title={t.mode.row} sub={interfaceMode === null ? '' : t.mode[interfaceMode].title} onClick={() => setSwitching(true)} />
        <Row icon="palette" tint="var(--g3)" title={t.theme.open} sub={t.theme.names[appearance]} onClick={() => setTheme(true)} />
        <Row icon="globe" tint="var(--g2)" title={t.more.myCurrencies} sub={`${currencies.length}`} onClick={() => setChoosing(true)} />
        <Row icon="globe" tint="var(--g4)" title={t.more.displayCurrency} sub={t.rates.countInHint} end={<DisplayCurrency display={display} currencies={currencies} onChange={onDisplay} />} />
        <label className="more-row">
          <span className="avatar" style={{ '--gc': 'var(--g2)' } as CSSProperties} aria-hidden="true">
            <svg viewBox="0 0 24 24">
              <path d={ICONS.globe} />
            </svg>
          </span>
          <span className="more-mid">
            <span className="more-title">{t.more.coinsInRates}</span>
            <span className="more-sub">{t.more.coinsInRatesHint}</span>
          </span>
          <span className="switch">
            <input type="checkbox" checked={coinsInRates} onChange={(e) => onCoinsInRates(e.target.checked)} />
            <span className="switch-track" />
          </span>
        </label>
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
        <Row icon="download" tint="var(--g4)" title={t.accounts.export} onClick={() => setExporting(true)} />
      </div>

      <h2 className="day">{t.more.help}</h2>
      <div className="card list-card">
        <Row icon="bell" tint="var(--g1)" title={t.accounts.reminder} sub={reminderHour === null ? t.more.off : `${String(reminderHour).padStart(2, '0')}:00`} onClick={() => setReminding(true)} />
        <Row icon="help" tint="var(--g6)" title={t.accounts.tour} onClick={() => void setTourSeen(false).then(onChanged, () => undefined)} />
      </div>

      {switching && (
        <Modal title={t.mode.title} onClose={() => setSwitching(false)}>
          <ModePicker
            value={interfaceMode}
            onPick={(m) => {
              void setInterfaceMode(m).then(() => {
                setSwitching(false)
                onChanged()
              }, () => undefined)
            }}
          />
        </Modal>
      )}
      {choosing && <MyCurrencies registry={registry} enabled={enabled ?? currencies} pinned={pinned} onChange={onEnabled} onClose={() => setChoosing(false)} />}
      {theme && <ThemeSettings onClose={() => setTheme(false)} />}
      {exporting && <ExportScreen onClose={() => setExporting(false)} />}
      {reminding && <ReminderSettings hour={reminderHour} onChanged={onChanged} onClose={() => setReminding(false)} />}
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
