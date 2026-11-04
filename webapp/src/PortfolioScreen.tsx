import { useEffect, useState, type CSSProperties } from 'react'
import { fetchPortfolio, type Account, type Portfolio } from './api'
import { AddHoldingForm } from './AddHoldingForm'
import { DisplayCurrency } from './DisplayCurrency'
import { Modal } from './Modal'
import { t } from './i18n'
import { Money } from './MoneyText'
import { currencySign, formatRate } from './money'
import { Segmented } from './Pickers'
import { formatBP, heatTone, shares, treemap } from './portfolio'
import { CIRCLE, RADIUS, arcs } from './report'

type Props = {
  display: string
  currencies: string[]
  wallets: Account[]
  today: string
  // Кошельки и записи изменились: данные приложения нужно перечитать.
  onChanged: () => void
  onDisplay: (code: string) => void
  onBack: () => void
}

type Loaded = { kind: 'loading' } | { kind: 'failed' } | { kind: 'ready'; portfolio: Portfolio }
type View = 'ring' | 'heat'

// Размер поля тепловой карты в условных единицах: плитки считаются в них, а на экране занимают проценты поля.
const HEAT_W = 340
const HEAT_H = 300

const palette = ['#0e7c7b', '#4c5bd4', '#d9892b', '#d4577a', '#7a8794', '#5fb3b0', '#7c88e0', '#a9b1ec', '#cdd2f5', '#a5d3d1']

const viewOptions = [
  { value: 'ring' as const, label: t.portfolio.ring },
  { value: 'heat' as const, label: t.portfolio.heat },
]

// Портфель: монеты во всех кошельках. Кольцо показывает доли по стоимости, тепловая карта прибыль в процентах от цены
// покупки: площадь плитки это доля, цвет это прибыль.
export function PortfolioScreen({ display, currencies, wallets, today, onChanged, onDisplay, onBack }: Props) {
  const [loaded, setLoaded] = useState<Loaded>({ kind: 'loading' })
  const [view, setView] = useState<View>('ring')
  const [adding, setAdding] = useState(false)
  const [picked, setPicked] = useState<string | null>(null)
  // Растёт после добавления: портфель запрашивается заново.
  const [version, setVersion] = useState(0)

  useEffect(() => {
    let alive = true
    fetchPortfolio(display)
      .then((portfolio) => alive && setLoaded({ kind: 'ready', portfolio }))
      .catch(() => alive && setLoaded({ kind: 'failed' }))
    return () => {
      alive = false
    }
  }, [display, version])

  const items = loaded.kind === 'ready' ? loaded.portfolio.items.filter((i) => i.quantity_minor > 0) : []
  const total = items.reduce((s, i) => s + i.value_minor, 0)
  const drawn = arcs(items.map((i) => i.value_minor))
  const percents = shares(items)
  const cur = loaded.kind === 'ready' ? loaded.portfolio.currency : display
  const current = items.find((i) => i.currency === picked)
  const tiles = treemap(
    items.map((i) => i.value_minor),
    HEAT_W,
    HEAT_H,
  )

  function pick(code: string) {
    setPicked((now) => (now === code ? null : code))
  }

  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onBack}>
          {t.back}
        </button>
        <span className="row-tight">
          <DisplayCurrency display={display} currencies={currencies} onChange={onDisplay} />
          <button type="button" className="chip" onClick={() => setAdding(true)}>
            + {t.portfolioAdd.open}
          </button>
        </span>
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
                    {drawn.map((a, i) => {
                      const item = items[i]
                      const active = item?.currency === picked
                      const mid = ((-a.offset + a.length / 2) / CIRCLE) * 2 * Math.PI
                      return (
                        <circle
                          key={item?.currency}
                          className="donut-seg donut-seg-tap"
                          cx="100"
                          cy="100"
                          r={RADIUS}
                          fill="none"
                          stroke={palette[i % palette.length]}
                          strokeWidth={active ? 30 : 24}
                          strokeDasharray={`${a.length} 1000`}
                          strokeDashoffset={a.offset}
                          style={{ transform: active ? `translate(${(Math.cos(mid) * 6).toFixed(2)}px, ${(Math.sin(mid) * 6).toFixed(2)}px)` : undefined, opacity: picked === null || active ? 1 : 0.35 }}
                          role="button"
                          tabIndex={0}
                          aria-label={item?.currency}
                          aria-pressed={active}
                          onClick={() => item !== undefined && pick(item.currency)}
                          onKeyDown={(e) => {
                            if ((e.key === 'Enter' || e.key === ' ') && item !== undefined) {
                              e.preventDefault()
                              pick(item.currency)
                            }
                          }}
                        />
                      )
                    })}
                  </g>
                </svg>
                <div className="donut-center">
                  <span className="donut-label">{current === undefined ? t.portfolio.total : current.currency}</span>
                  <Money minor={current === undefined ? total : current.value_minor} currency={cur} />
                  {current !== undefined && <span className="donut-pct">{percents[items.indexOf(current)]}%</span>}
                </div>
              </div>
            ) : (
              <>
                <div className="heat" role="group" aria-label={t.portfolio.heat}>
                  {items.map((i, k) => {
                    const r = tiles[k]
                    if (r === undefined || r.w === 0 || r.h === 0) return null
                    const { tone, strength } = heatTone(i.profit_bp)
                    const small = r.w < HEAT_W * 0.2 || r.h < HEAT_H * 0.24
                    const active = i.currency === picked
                    const style = {
                      left: `${(r.x / HEAT_W) * 100}%`,
                      top: `${(r.y / HEAT_H) * 100}%`,
                      width: `${(r.w / HEAT_W) * 100}%`,
                      height: `${(r.h / HEAT_H) * 100}%`,
                      '--ts': strength,
                    } as CSSProperties
                    return (
                      <div
                        key={i.currency}
                        className={`heat-tile heat-${tone}${small ? ' heat-small' : ''}${active ? ' heat-on' : ''}`}
                        style={style}
                        role="button"
                        tabIndex={0}
                        aria-pressed={active}
                        onClick={() => pick(i.currency)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault()
                            pick(i.currency)
                          }
                        }}
                      >
                        <span className="heat-tk">{i.currency}</span>
                        <span>
                          <span className="heat-pct">{i.profit_bp === null ? '–' : formatBP(i.profit_bp)}</span>
                          <span className="heat-val">
                            <Money minor={i.value_minor} currency={cur} />
                          </span>
                        </span>
                      </div>
                    )
                  })}
                </div>
                <p className="hint tight">{t.portfolio.heatHint}</p>
              </>
            )}
          </div>

          <div className="card list-card">
            {items.map((i, idx) => (
              <div key={i.currency} className={i.currency === picked ? 'holding holding-on' : 'holding'} onClick={() => pick(i.currency)}>
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
          <p className="hint tight">{t.portfolio.rateNote}</p>
        </>
      )}

      {adding && (
        <Modal title={t.portfolioAdd.title} onClose={() => setAdding(false)}>
          <AddHoldingForm
            wallets={wallets}
            allCurrencies={currencies}
            today={today}
            onAssetAdded={onChanged}
            onDone={() => {
              setAdding(false)
              setVersion((v) => v + 1)
              onChanged()
            }}
          />
        </Modal>
      )}
    </main>
  )
}
