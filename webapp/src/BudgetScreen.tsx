import { useEffect, useState, type CSSProperties, type FormEvent } from 'react'
import {
  ApiError,
  createCategory,
  createCategoryGroup,
  fetchBudgetMonth,
  setBudgeted,
  setEnvelopes,
  updateCategory,
  type BudgetMonth,
  type Account,
  type BudgetRow,
  type Category,
  type CategoryGroup,
  type Transaction,
} from './api'
import { fillPercent, monthPace, parseBudgeted, rowStatus, spendingOf, toBudgetTone, type RowStatus } from './budget'
import { BudgetCompare } from './BudgetCompare'
import { BudgetFocus } from './BudgetFocus'
import { BudgetSuggest } from './BudgetSuggest'
import { CategoryRing } from './CategoryRing'
import { monthTitle, shiftMonth } from './calendar'
import { groupColors } from './groupColor'
import { errorText, t } from './i18n'
import { MonthNav } from './MonthNav'
import { Modal } from './Modal'
import { Money } from './MoneyText'
import { formatInput } from './money'
import { ChipPicker } from './Pickers'
import { percent } from './report'

type Props = {
  groups: CategoryGroup[]
  categories: Category[]
  accounts: Account[]
  // Валюта бюджета: все суммы здесь в ней.
  baseCurrency: string
  today: string
  // Включён ли режим конвертов: заложенные суммы, «К распределению», лимиты. Иначе только траты по категориям.
  envelopesEnabled: boolean
  // Записи бюджета: после их изменения бюджет месяца пересчитывается.
  ledger: Transaction[]
  onChanged: () => void
  onOpenCategory: (id: number) => void
  onOpenSchedules: () => void
}

type View = 'spending' | 'envelopes'

const EMPTY_ROW: BudgetRow ={ category_id: 0, budgeted_minor: 0, spent_minor: 0, balance_minor: 0 }

// Бюджет месяца: сколько осталось распределить и по каждой категории заложено, потрачено, остаток.
// Здесь же создаются и правятся категории и группы.
export function BudgetScreen({ groups, categories, accounts, baseCurrency, today, envelopesEnabled, ledger, onChanged, onOpenCategory, onOpenSchedules }: Props) {
  const [month, setMonth] = useState(today.slice(0, 7))
  const [view, setView] = useState<View>(envelopesEnabled ? 'envelopes' : 'spending')
  const [envelopesDialog, setEnvelopesDialog] = useState<'enable' | 'disable' | null>(null)
  const [envelopesError, setEnvelopesError] = useState<string | null>(null)
  const [previous, setPrevious] = useState<BudgetMonth | null>(null)
  const [budget, setBudget] = useState<BudgetMonth | null>(null)
  const [failed, setFailed] = useState(false)
  const [editMode, setEditMode] = useState(false)
  const [creating, setCreating] = useState(false)
  const [creatingGroup, setCreatingGroup] = useState(false)
  const [editing, setEditing] = useState<Category | null>(null)
  const [planning, setPlanning] = useState<Category | null>(null)
  const [adding, setAdding] = useState(false)
  const [focus, setFocus] = useState(false)
  const [comparing, setComparing] = useState(false)
  const [suggesting, setSuggesting] = useState(false)
  // Свёрнутые группы и группы, где показаны категории без сумм.
  const [collapsed, setCollapsed] = useState<Set<number>>(new Set())
  const [withEmpty, setWithEmpty] = useState<Set<number>>(new Set())

  // Месяц, записи и категории могли измениться: бюджет запрашивается заново, а старый остаётся на экране до ответа.
  useEffect(() => {
    let alive = true
    setFailed(false)
    fetchBudgetMonth(month)
      .then((b) => alive && setBudget(b))
      .catch(() => alive && setFailed(true))
    return () => {
      alive = false
    }
  }, [month, categories, ledger])

  // В режиме трат рядом с месяцем показано изменение к прошлому.
  useEffect(() => {
    if (view !== 'spending') return
    let alive = true
    fetchBudgetMonth(shiftMonth(month, -1))
      .then((b) => alive && setPrevious(b))
      .catch(() => alive && setPrevious(null))
    return () => {
      alive = false
    }
  }, [view, month, categories, ledger])

  // Режим конвертов выключили или включили не здесь: экран следует за флагом бюджета.
  useEffect(() => {
    setView(envelopesEnabled ? 'envelopes' : 'spending')
  }, [envelopesEnabled])

  const envelopes = view === 'envelopes'
  const spending = budget === null ? null : spendingOf(budget, previous)

  async function switchEnvelopes(enabled: boolean) {
    setEnvelopesError(null)
    try {
      await setEnvelopes(enabled)
      setEnvelopesDialog(null)
      setView(enabled ? 'envelopes' : 'spending')
      // Сразу после включения предлагаем не заполнять суммы руками.
      if (enabled) setSuggesting(true)
      onChanged()
    } catch (e) {
      setEnvelopesError(errorText(e instanceof ApiError ? e.code : 'internal'))
    }
  }

  const groupOptions = groups.map((g) => ({ value: g.id, label: g.name }))
  const firstGroup = groups[0]?.id ?? 0
  const rows = new Map((budget?.categories ?? []).map((r) => [r.category_id, r]))
  const tone = budget === null ? 'done' : toBudgetTone(budget.to_budget_minor)
  const pace = monthPace(month, today)
  const colorOf = groupColors(groups)
  const groupSum = (groupId: number, key: 'budgeted_minor' | 'spent_minor') =>
    categories.filter((c) => c.group_id === groupId).reduce((sum, c) => sum + (rows.get(c.id)?.[key] ?? 0), 0)
  // Категории, на которые стоит посмотреть: перерасход, траты без плана, опережение темпа. Только в режиме конвертов.
  const rank: Partial<Record<RowStatus, number>> = { over: 0, noplan: 1, warn: 2 }
  const attention =
    !envelopes || budget === null || editMode
      ? []
      : categories
          .filter((c) => !c.hidden)
          .map((c) => ({ c, r: rows.get(c.id) ?? EMPTY_ROW }))
          .map((x) => ({ ...x, st: rowStatus(x.r, pace) }))
          .filter((x) => rank[x.st] !== undefined)
          .sort((a, b) => (rank[a.st] ?? 0) - (rank[b.st] ?? 0))
  const plannedBase = budget === null ? 0 : Math.max(budget.income_minor, budget.budgeted_minor)

  function flip(set: Set<number>, id: number): Set<number> {
    const next = new Set(set)
    if (!next.delete(id)) next.add(id)
    return next
  }

  return (
    <main className="screen">
      <header className="row">
        <h1>{t.budget.title}</h1>
        <button type="button" className="plus" aria-label={t.schedules.open} title={t.schedules.open} onClick={onOpenSchedules}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <rect x="4" y="5" width="16" height="15" rx="3" />
            <path d="M8 3v4M16 3v4M4 10h16" />
          </svg>
        </button>
      </header>

      <MonthNav title={monthTitle(month)} onPrev={() => setMonth(shiftMonth(month, -1))} onNext={() => setMonth(shiftMonth(month, 1))} />

      <div className="field">
        <div className="chips" role="radiogroup" aria-label={t.budget.title}>
          <button type="button" role="radio" aria-checked={!envelopes} className={!envelopes ? 'pill pill-on' : 'pill'} onClick={() => setView('spending')}>
            {t.budget.views.spending}
          </button>
          <button
            type="button"
            role="radio"
            aria-checked={envelopes}
            className={envelopes ? 'pill pill-on' : 'pill'}
            onClick={() => (envelopesEnabled ? setView('envelopes') : setEnvelopesDialog('enable'))}
          >
            {envelopesEnabled ? t.budget.views.envelopes : t.budget.views.enable}
          </button>
        </div>
      </div>

      {failed && <p className="error">{t.errors.internal}</p>}
      {!envelopes && spending !== null && (
        <div className="card budget-head budget-hero">
          <span className="label">{t.budget.spendingTotal}</span>
          <Money minor={spending.totalMinor} currency={baseCurrency} />
          {spending.previousTotalMinor > 0 && spending.totalMinor !== spending.previousTotalMinor && (
            <p className="hint tight">
              {t.budget.vsPrevious.by} <Money minor={Math.abs(spending.totalMinor - spending.previousTotalMinor)} currency={baseCurrency} />{' '}
              {spending.totalMinor > spending.previousTotalMinor ? t.budget.vsPrevious.more : t.budget.vsPrevious.less}
            </p>
          )}
          {spending.totalMinor > 0 && (
            <>
              <div className="split" aria-hidden="true">
                {groups.map((g) => {
                  const spent = groupSum(g.id, 'spent_minor')
                  return spent > 0 ? <span key={g.id} className="split-part" style={{ width: `${(spent / spending.totalMinor) * 100}%`, background: colorOf.get(g.id) }} /> : null
                })}
              </div>
              <div className="split-legend">
                {groups.map((g) => {
                  const spent = groupSum(g.id, 'spent_minor')
                  return spent > 0 ? (
                    <span key={g.id}>
                      <i style={{ background: colorOf.get(g.id) }} />
                      {g.name} <Money minor={spent} currency={baseCurrency} />
                    </span>
                  ) : null
                })}
              </div>
            </>
          )}
          {budget !== null && budget.missing_rates > 0 && (
            <p className="hint tight">
              {t.reports.missingRates}: {budget.missing_rates}
            </p>
          )}
        </div>
      )}
      {envelopes && budget !== null && (
        <div className={`card budget-head budget-hero budget-tap budget-${tone}`} role="button" tabIndex={0} onClick={() => setFocus(true)} onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && setFocus(true)}>
          <span className="label">{t.budget.toBudget}</span>
          <Money minor={budget.to_budget_minor} currency={baseCurrency} />
          <p className="budget-tone">{t.budget.tone[tone]}</p>
          {plannedBase > 0 && (
            <>
              <div className="split" aria-hidden="true">
                {groups.map((g) => {
                  const planned = groupSum(g.id, 'budgeted_minor')
                  return planned > 0 ? <span key={g.id} className="split-part" style={{ width: `${(planned / plannedBase) * 100}%`, background: colorOf.get(g.id) }} /> : null
                })}
                <span className="split-free" />
              </div>
              <div className="split-legend">
                {groups.map((g) => {
                  const planned = groupSum(g.id, 'budgeted_minor')
                  return planned > 0 ? (
                    <span key={g.id}>
                      <i style={{ background: colorOf.get(g.id) }} />
                      {g.name} <Money minor={planned} currency={baseCurrency} />
                    </span>
                  ) : null
                })}
              </div>
            </>
          )}
          <div className="budget-sub">
            <span>
              {t.budget.income} <Money minor={budget.income_minor} currency={baseCurrency} />
            </span>
            <span>
              {t.budget.budgeted} <Money minor={budget.budgeted_minor} currency={baseCurrency} />
            </span>
          </div>
          {budget.missing_rates > 0 && (
            <p className="hint tight">
              {t.reports.missingRates}: {budget.missing_rates}
            </p>
          )}
        </div>
      )}

      {attention.length > 0 && (
        <>
          <h2 className="day">
            {t.budget.attention} · {attention.length}
          </h2>
          <div className="attn">
            {attention.map(({ c, r, st }) => (
              <div key={c.id} className="attn-chip" role="button" tabIndex={0} onClick={() => onOpenCategory(c.id)} onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && onOpenCategory(c.id)}>
                <i className={`attn-dot attn-${st}`} />
                {c.name}
                <em>{st === 'noplan' ? t.budget.caption.noplan : <Money minor={r.balance_minor} currency={baseCurrency} />}</em>
              </div>
            ))}
          </div>
        </>
      )}

      <div className="toolbar">
        <button type="button" className={editMode ? 'chip chip-on' : 'chip'} onClick={() => setEditMode(!editMode)}>
          {editMode ? t.categories.editDone : t.categories.edit}
        </button>
        <button type="button" className="chip" onClick={() => setAdding(true)}>
          + {t.budget.add}
        </button>
        <button type="button" className="chip" disabled={budget === null} onClick={() => setComparing(true)}>
          {t.budget.compare.open}
        </button>
        {envelopes && (
          <button type="button" className="chip" onClick={() => setSuggesting(true)}>
            {t.budget.suggest.open}
          </button>
        )}
      </div>

      {groups.map((g) => {
        const isActive = (c: Category) => {
          const r = rows.get(c.id) ?? EMPTY_ROW
          return envelopes ? r.budgeted_minor !== 0 || r.spent_minor !== 0 || r.balance_minor !== 0 : r.spent_minor > 0
        }
        const visible = categories.filter((c) => c.group_id === g.id && (editMode || !c.hidden || isActive(c)))
        // Категории без сумм прячутся за строкой «Без сумм», пока бюджет не загружен, показываются все.
        const emptyCount = budget === null || editMode ? 0 : visible.filter((c) => !isActive(c)).length
        const showEmpty = emptyCount === 0 || withEmpty.has(g.id)
        const inGroup = collapsed.has(g.id) ? [] : visible.filter((c) => showEmpty || isActive(c))
        const gc = colorOf.get(g.id)
        const toggle = () => setCollapsed(flip(collapsed, g.id))
        return (
          <section key={g.id} style={{ '--gc': gc } as CSSProperties}>
            <div className="b2-gh" role="button" tabIndex={0} aria-expanded={!collapsed.has(g.id)} onClick={toggle} onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && toggle()}>
              <i />
              <span className="gn">{g.name}</span>
              {budget !== null && (
                <span className="gs">
                  <Money minor={groupSum(g.id, 'spent_minor')} currency={baseCurrency} />
                  {envelopes && (
                    <>
                      {' '}
                      {t.budget.of} <Money minor={groupSum(g.id, 'budgeted_minor')} currency={baseCurrency} />
                    </>
                  )}
                </span>
              )}
              <span className={collapsed.has(g.id) ? 'chev' : 'chev chev-open'}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M9 6l6 6-6 6" />
                </svg>
              </span>
            </div>
            <div className={collapsed.has(g.id) ? 'card list-card list-folded' : 'card list-card'}>
              {!collapsed.has(g.id) && visible.length === 0 && <p className="hint tight pad">{t.categories.empty}</p>}
              {inGroup.map((c) => {
                const r = rows.get(c.id) ?? EMPTY_ROW
                const was = spending?.rows.find((x) => x.categoryId === c.id)?.previousMinor ?? 0
                const st: RowStatus = envelopes ? rowStatus(r, pace) : 'ok'
                const share = percent(r.spent_minor, spending?.totalMinor ?? 0)
                const open = () => (editMode ? setEditing(c) : onOpenCategory(c.id))
                return (
                  <div key={c.id} className={`b2-row s-${st}${c.hidden ? ' entry-muted' : ''}`} role="button" tabIndex={0} onClick={open} onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && open()}>
                    <CategoryRing name={c.name} fill={envelopes ? fillPercent(r.spent_minor, r.budgeted_minor) : share} pace={envelopes && r.budgeted_minor > 0 ? pace : null} status={st} />
                    <div className="b2-mid">
                      <span className="b2-name">
                        {c.name}
                        {c.hidden && <span className="badge-muted">{t.categories.hidden}</span>}
                      </span>
                      {envelopes ? (
                        <span className="b2-sub">
                          <b>
                            <Money minor={r.spent_minor} currency={baseCurrency} />
                          </b>
                          <span
                            className="b2-plan"
                            role="button"
                            tabIndex={editMode ? -1 : 0}
                            aria-disabled={editMode}
                            onClick={(e) => {
                              e.stopPropagation()
                              if (!editMode) setPlanning(c)
                            }}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter' || e.key === ' ') {
                                e.stopPropagation()
                                if (!editMode) setPlanning(c)
                              }
                            }}
                          >
                            {r.budgeted_minor > 0 ? (
                              <>
                                {t.budget.of} <Money minor={r.budgeted_minor} currency={baseCurrency} />
                              </>
                            ) : (
                              t.budget.setPlan
                            )}
                          </span>
                          {c.rollover && (
                            <span className="b2-roll" title={t.budget.rolls} aria-label={t.budget.rolls}>
                              ↻
                            </span>
                          )}
                        </span>
                      ) : (
                        r.spent_minor > 0 && (
                          <span className="b2-sub">
                            {share}% {t.budget.ofTotal}
                            {previous !== null && r.spent_minor !== was && (
                              <span>
                                <Money minor={r.spent_minor - was} currency={baseCurrency} signed /> {t.budget.toPrevious}
                              </span>
                            )}
                          </span>
                        )
                      )}
                    </div>
                    <div className="b2-end">
                      <Money minor={envelopes ? r.balance_minor : r.spent_minor} currency={baseCurrency} />
                      <small>{envelopes ? t.budget.caption[st] : t.budget.spent}</small>
                    </div>
                    {editMode && (
                      <span className="edit-mark" aria-hidden="true">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <path d="M4 20h4L19 9l-4-4L4 16zM13 7l4 4" />
                        </svg>
                      </span>
                    )}
                  </div>
                )
              })}
              {!collapsed.has(g.id) && emptyCount > 0 && (
                <button type="button" className="empty-rows" onClick={() => setWithEmpty(flip(withEmpty, g.id))}>
                  {showEmpty ? t.budget.collapse : `${t.budget.emptyRows}: ${emptyCount}`}
                </button>
              )}
            </div>
          </section>
        )
      })}

      {focus && budget !== null && <BudgetFocus budget={budget} tone={tone} accounts={accounts} categories={categories} baseCurrency={baseCurrency} onClose={() => setFocus(false)} />}

      {envelopes && (
        <button type="button" className="chip" onClick={() => setEnvelopesDialog('disable')}>
          {t.budget.envelopes.disable}
        </button>
      )}

      {envelopesDialog === 'enable' && (
        <Modal title={t.budget.envelopes.title} onClose={() => setEnvelopesDialog(null)}>
          <p className="hint">{t.budget.envelopes.intro}</p>
          <ul className="hint">
            {t.budget.envelopes.points.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
          {envelopesError !== null && <p className="error">{envelopesError}</p>}
          <button type="button" className="primary" onClick={() => void switchEnvelopes(true)}>
            {t.budget.envelopes.enable}
          </button>
        </Modal>
      )}
      {envelopesDialog === 'disable' && (
        <Modal title={t.budget.envelopes.disableTitle} onClose={() => setEnvelopesDialog(null)}>
          <p className="hint">{t.budget.envelopes.disableHint}</p>
          {envelopesError !== null && <p className="error">{envelopesError}</p>}
          <button type="button" className="primary" onClick={() => void switchEnvelopes(false)}>
            {t.budget.envelopes.disable}
          </button>
        </Modal>
      )}

      {comparing && budget !== null && <BudgetCompare month={month} budget={budget} categories={categories} baseCurrency={baseCurrency} plan={envelopes} onClose={() => setComparing(false)} />}
      {suggesting && (
        <BudgetSuggest
          month={month}
          current={new Map([...rows].map(([id, r]) => [id, r.budgeted_minor]))}
          categories={categories}
          baseCurrency={baseCurrency}
          onApplied={(b) => {
            setBudget(b)
            setSuggesting(false)
          }}
          onClose={() => setSuggesting(false)}
        />
      )}

      {adding && (
        <Modal title={t.budget.add} onClose={() => setAdding(false)}>
          <div className="stack">
            <button
              type="button"
              className="option"
              onClick={() => {
                setAdding(false)
                setCreating(true)
              }}
            >
              {t.categories.new}
            </button>
            <button
              type="button"
              className="option"
              onClick={() => {
                setAdding(false)
                setCreatingGroup(true)
              }}
            >
              {t.categories.newGroup}
            </button>
          </div>
        </Modal>
      )}

      {planning !== null && (
        <Modal title={planning.name} onClose={() => setPlanning(null)}>
          <PlanForm
            category={planning}
            month={month}
            currency={baseCurrency}
            current={rows.get(planning.id)?.budgeted_minor ?? 0}
            onSaved={(b) => {
              setBudget(b)
              setPlanning(null)
            }}
          />
        </Modal>
      )}

      {creating && (
        <Modal title={t.categories.new} onClose={() => setCreating(false)}>
          <CategoryForm
            initial={{ name: '', description: '', groupId: firstGroup, hidden: false, rollover: false }}
            groupOptions={groupOptions}
            onSubmit={async (v) => {
              await createCategory(v.name, v.groupId)
              setCreating(false)
              onChanged()
            }}
          />
        </Modal>
      )}

      {creatingGroup && (
        <Modal title={t.categories.newGroup} onClose={() => setCreatingGroup(false)}>
          <GroupForm
            onSubmit={async (name) => {
              await createCategoryGroup(name)
              setCreatingGroup(false)
              onChanged()
            }}
          />
        </Modal>
      )}

      {editing !== null && (
        <Modal title={t.categories.editTitle} onClose={() => setEditing(null)}>
          <CategoryForm
            initial={{ name: editing.name, description: editing.description, groupId: editing.group_id, hidden: editing.hidden, rollover: editing.rollover }}
            withDetails
            onSubmit={async (v) => {
              const patch: { name?: string; description?: string; hidden?: boolean; rollover?: boolean } = {}
              if (v.name !== editing.name) patch.name = v.name
              if (v.description !== editing.description) patch.description = v.description
              if (v.hidden !== editing.hidden) patch.hidden = v.hidden
              if (v.rollover !== editing.rollover) patch.rollover = v.rollover
              if (Object.keys(patch).length > 0) await updateCategory(editing.id, patch)
              setEditing(null)
              onChanged()
            }}
          />
        </Modal>
      )}
    </main>
  )
}

// useSubmit держит общее состояние форм: занято, текст ошибки сервера.
function useSubmit() {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function run(e: FormEvent, action: () => Promise<void>) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await action()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }
  return { busy, error, run, fail: setError }
}

type PlanProps = {
  category: Category
  month: string
  currency: string
  current: number
  onSaved: (b: BudgetMonth) => void
}

// PlanForm: сколько заложить на категорию в этом месяце. Пусто или ноль снимают заложенное.
function PlanForm({ category, month, currency, current, onSaved }: PlanProps) {
  const [amount, setAmount] = useState(current > 0 ? formatInput(current, currency) : '')
  const { busy, error, run, fail } = useSubmit()

  return (
    <form
      className="form"
      onSubmit={(e) =>
        void run(e, async () => {
          const minor = parseBudgeted(amount, currency)
          if (minor === null) {
            fail(errorText('invalid_amount'))
            return
          }
          onSaved(await setBudgeted(month, category.id, minor))
        })
      }
    >
      <label className="field">
        <span className="label">
          {t.budget.planFor} {monthTitle(month)}
        </span>
        <input inputMode="decimal" value={amount} placeholder="0,00" autoFocus onChange={(e) => setAmount(e.target.value)} />
      </label>
      <p className="hint tight">{category.rollover ? t.budget.rollsHint : t.budget.noRollHint}</p>
      {error !== null && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy}>
        {t.save}
      </button>
    </form>
  )
}

function GroupForm({ onSubmit }: { onSubmit: (name: string) => Promise<void> }) {
  const [name, setName] = useState('')
  const { busy, error, run } = useSubmit()

  return (
    <form className="form" onSubmit={(e) => void run(e, () => onSubmit(name))}>
      <label className="field">
        <span className="label">{t.categories.name}</span>
        <input value={name} maxLength={40} autoFocus onChange={(e) => setName(e.target.value)} />
      </label>
      {error !== null && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy || name.trim() === ''}>
        {t.save}
      </button>
    </form>
  )
}

type Values = {
  name: string
  description: string
  groupId: number
  hidden: boolean
  rollover: boolean
}

type FormProps = {
  initial: Values
  // groupOptions: выбор группы при создании. withDetails: описание, «скрыта» и перенос при правке.
  groupOptions?: { value: number; label: string }[]
  withDetails?: boolean
  onSubmit: (v: Values) => Promise<void>
}

function CategoryForm({ initial, groupOptions, withDetails = false, onSubmit }: FormProps) {
  const [v, setV] = useState(initial)
  const { busy, error, run } = useSubmit()

  return (
    <form className="form" onSubmit={(e) => void run(e, () => onSubmit(v))}>
      <label className="field">
        <span className="label">{t.categories.name}</span>
        <input value={v.name} maxLength={40} autoFocus onChange={(e) => setV({ ...v, name: e.target.value })} />
      </label>
      {groupOptions !== undefined && (
        <ChipPicker label={t.categories.group} value={v.groupId} options={groupOptions} onChange={(groupId) => setV({ ...v, groupId })} />
      )}
      {withDetails && (
        <>
          <label className="field">
            <span className="label">{t.categories.description}</span>
            <input value={v.description} maxLength={200} onChange={(e) => setV({ ...v, description: e.target.value })} />
          </label>
          <label className="switch">
            <input type="checkbox" checked={v.rollover} onChange={(e) => setV({ ...v, rollover: e.target.checked })} />
            <span className="switch-track" aria-hidden="true" />
            <span>{t.categories.rollover}</span>
          </label>
          <p className="hint tight">{t.categories.rolloverHint}</p>
          <label className="switch">
            <input type="checkbox" checked={v.hidden} onChange={(e) => setV({ ...v, hidden: e.target.checked })} />
            <span className="switch-track" aria-hidden="true" />
            <span>{t.categories.hidden}</span>
          </label>
        </>
      )}
      {error !== null && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy || v.name.trim() === ''}>
        {t.save}
      </button>
    </form>
  )
}
