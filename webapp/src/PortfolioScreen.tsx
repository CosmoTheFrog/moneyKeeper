import { useEffect, useState } from 'react'
import { fetchPortfolio, type Portfolio } from './api'
import { DisplayCurrency } from './DisplayCurrency'
import { t } from './i18n'
import { Money } from './MoneyText'
import { currencySign, formatRate } from './money'
import { Segmented } from './Pickers'
import { bubbleSize, formatBP, shares } from './portfolio'
import { RADIUS, arcs } from './report'

type Props = {
  display: string
  currencies: string[]
  onDisplay: (code: string) => void
  onBack: () => void
}

type Loaded = { kind: 'loading' } | { kind: 'failed' } | { kind: 'ready'; portfolio: Portfolio }
type View = 'ring' | 'bubbles'

const palette = ['#0e7c7b', '#4c5bd4', '#d9892b', '#d4577a', '#7a8794', '#5fb3b0', '#7c88e0', '#a9b1ec', '#cdd2f5', '#a5d3d1']

const viewOptions = [
  { value: 'ring' as const, label: t.portfolio.ring },
  { value: 'bubbles' as const, label: t.portfolio.bubbles },
]

// Портфель: монеты во всех кошельках. Кольцо показывает доли по стоимости, пузырьки прибыль в процентах от цены покупки.
export function PortfolioScreen({ display, currencies, onDisplay, onBack }: Props) {
  const [loaded, setLoaded] = useState<Loaded>({ kind: 'loading' })
  const [view, setView] = useState<View>('ring')

  useEffect(() => {
    let alive = true
    fetchPortfolio(display)
      .then((portfolio) => alive && setLoaded({ kind: 'ready', portfolio }))
      .catch(() => alive && setLoaded({ kind: 'failed' }))
    return () => {
      alive = false
    }
  }, [display])

  const items = loaded.kind === 'ready' ? loaded.portfolio.items.filter((i) => i.quantity_minor > 0) : []
  const total = items.reduce((s, i) => s + i.value_minor, 0)
  const drawn = arcs(items.map((i) => i.value_minor))
  const percents = shares(items)
  const cur = loaded.kind === 'ready' ? loaded.portfolio.currency : display

  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onBack}>
          {t.back}
        </button>
        <DisplayCurrency display={display} currencies={currencies} onChange={onDisplay} />
      </header>
      <h1>{t.portfolio.title}</h1>

      {loaded.kind === 'loading' && <p className="hint">{t.loading}</p>}
      {loaded.kind === 'failed' && <p className="error">{t.errors.internal}</p>}
      {loaded.kind === 'ready' && items.length === 0 && <p className="hint">{t.portfolio.empty}</p>}

      {loaded.kind === 'ready' && items.length > 0 && (
        <>
          <Segmented value={view} options={viewOptions} onChange={setView} />

          <div className="card">
            {view === 'ring' ? (
              <div className="donut">
                <svg viewBox="0 0 200 200" role="img" aria-label={t.portfolio.ring}>
                  <g transform="rotate(-90 100 100)">
                    <circle cx="100" cy="100" r={RADIUS} fill="none" style={{ stroke: 'var(--line)' }} strokeWidth="24" />
                    {drawn.map((a, i) => (
                      <circle key={items[i]?.currency} cx="100" cy="100" r={RADIUS} fill="none" stroke={palette[i % palette.length]} strokeWidth="24" strokeDasharray={`${a.length} 1000`} strokeDashoffset={a.offset} />
                    ))}
                  </g>
                </svg>
                <div className="donut-center">
                  <span className="donut-label">{t.portfolio.total}</span>
                  <Money minor={total} currency={cur} />
                </div>
              </div>
            ) : (
              <>
                <div className="bubbles">
                  {items.map((i) => {
                    const size = bubbleSize(i.profit_bp)
                    const up = (i.profit_bp ?? 0) >= 0
                    return (
                      <div key={i.currency} className={up ? 'bubble bubble-up' : 'bubble bubble-down'} style={{ width: size, height: size }}>
                        <b>{i.currency}</b>
                        <span>{i.profit_bp === null ? '–' : formatBP(i.profit_bp)}</span>
                      </div>
                    )
                  })}
                </div>
                <p className="hint tight">{t.portfolio.bubbleHint}</p>
              </>
            )}
          </div>

          <div className="card list-card">
            {items.map((i, idx) => (
              <div key={i.currency} className="holding">
                <div className="holding-top">
                  <span className="dot" style={{ background: palette[idx % palette.length] }} />
                  <b className="holding-name">{i.currency}</b>
                  <Money minor={i.value_minor} currency={cur} />
                  <span className="legend-pct">{percents[idx]}%</span>
                </div>
                <div className="holding-sub">
                  <span>
                    <Money minor={i.quantity_minor} currency={i.currency} />
                  </span>
                  {i.profit_bp !== null ? (
                    <span className={i.profit_minor >= 0 ? 'profit-up' : 'profit-down'}>
                      {t.portfolio.profit} <Money minor={i.profit_minor} currency={cur} signed /> · {formatBP(i.profit_bp)}
                    </span>
                  ) : (
                    <span>{t.portfolio.noCost}</span>
                  )}
                </div>
                {i.avg_price !== '' && (
                  <div className="holding-sub">
                    <span>
                      {t.portfolio.avg}: {formatRate(i.avg_price)} {currencySign(cur)}
                    </span>
                    {i.realized_minor !== 0 && (
                      <span>
                        {t.portfolio.realized} <Money minor={i.realized_minor} currency={cur} signed />
                      </span>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
          {loaded.portfolio.missing_rates > 0 && (
            <p className="hint tight">
              {t.reports.missingRates}: {loaded.portfolio.missing_rates}
            </p>
          )}
        </>
      )}
    </main>
  )
}
