import { useEffect, useState, type FormEvent } from 'react'
import {
  ApiError,
  createCategory,
  createCategoryGroup,
  fetchBudgetMonth,
  setBudgeted,
  updateCategory,
  type BudgetMonth,
  type BudgetRow,
  type Category,
  type CategoryGroup,
  type Transaction,
} from './api'
import { fillPercent, parseBudgeted, toBudgetTone } from './budget'
import { monthTitle, shiftMonth } from './calendar'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'
import { formatInput } from './money'
import { ChipPicker } from './Pickers'

type Props = {
  groups: CategoryGroup[]
  categories: Category[]
  // Валюта бюджета: все суммы здесь в ней.
  baseCurrency: string
  today: string
  // Записи бюджета: после их изменения бюджет месяца пересчитывается.
  ledger: Transaction[]
  onChanged: () => void
  onOpenCategory: (id: number) => void
  onAddTo: (id: number) => void
}

const EMPTY_ROW: BudgetRow = { category_id: 0, budgeted_minor: 0, spent_minor: 0, balance_minor: 0 }

function Chevron({ left }: { left: boolean }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={left ? 'M15 6l-6 6 6 6' : 'M9 6l6 6-6 6'} />
    </svg>
  )
}

// Бюджет месяца: сколько осталось распределить и по каждой категории заложено, потрачено, остаток.
// Здесь же создаются и правятся категории и группы.
export function BudgetScreen({ groups, categories, baseCurrency, today, ledger, onChanged, onOpenCategory, onAddTo }: Props) {
  const [month, setMonth] = useState(today.slice(0, 7))
  const [budget, setBudget] = useState<BudgetMonth | null>(null)
  const [failed, setFailed] = useState(false)
  const [editMode, setEditMode] = useState(false)
  const [creating, setCreating] = useState(false)
  const [creatingGroup, setCreatingGroup] = useState(false)
  const [editing, setEditing] = useState<Category | null>(null)
  const [planning, setPlanning] = useState<Category | null>(null)

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

  const groupOptions = groups.map((g) => ({ value: g.id, label: g.name }))
  const firstGroup = groups[0]?.id ?? 0
  const rows = new Map((budget?.categories ?? []).map((r) => [r.category_id, r]))
  const tone = budget === null ? 'done' : toBudgetTone(budget.to_budget_minor)

  return (
    <main className="screen">
      <h1>{t.budget.title}</h1>

      <div className="month-nav">
        <button type="button" className="plus" aria-label={t.calendar.prev} onClick={() => setMonth(shiftMonth(month, -1))}>
          <Chevron left />
        </button>
        <span className="month-title">{monthTitle(month)}</span>
        <button type="button" className="plus" aria-label={t.calendar.next} onClick={() => setMonth(shiftMonth(month, 1))}>
          <Chevron left={false} />
        </button>
      </div>

      {failed && <p className="error">{t.errors.internal}</p>}
      {budget !== null && (
        <div className={`card budget-head budget-${tone}`}>
          <span className="label">{t.budget.toBudget}</span>
          <Money minor={budget.to_budget_minor} currency={baseCurrency} />
          <p className="budget-tone">{t.budget.tone[tone]}</p>
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

      <div className="toolbar">
        <button type="button" className={editMode ? 'chip chip-on' : 'chip'} onClick={() => setEditMode(!editMode)}>
          {editMode ? t.categories.editDone : t.categories.edit}
        </button>
        <button type="button" className="chip" onClick={() => setCreating(true)}>
          {t.categories.new}
        </button>
        <button type="button" className="chip" onClick={() => setCreatingGroup(true)}>
          {t.categories.newGroup}
        </button>
      </div>

      {groups.map((g) => {
        const inGroup = categories.filter((c) => {
          const r = rows.get(c.id) ?? EMPTY_ROW
          const active = r.budgeted_minor !== 0 || r.spent_minor !== 0 || r.balance_minor !== 0
          return c.group_id === g.id && (editMode || !c.hidden || active)
        })
        return (
          <section key={g.id}>
            <h2 className="day">{g.name}</h2>
            <div className="card list-card">
              {inGroup.length === 0 && <p className="hint tight pad">{t.categories.empty}</p>}
              {inGroup.map((c) => {
                const r = rows.get(c.id) ?? EMPTY_ROW
                return (
                  <div key={c.id} className={c.hidden ? 'budget-row entry-muted' : 'budget-row'}>
                    <div className="budget-top">
                      <button type="button" className="entry-open" onClick={() => (editMode ? setEditing(c) : onOpenCategory(c.id))}>
                        <span className="entry-title">{c.name}</span>
                        <span className="entry-sub">{c.description}</span>
                      </button>
                      {c.hidden && <span className="badge-muted">{t.categories.hidden}</span>}
                      <span className="budget-balance">
                        <Money minor={r.balance_minor} currency={baseCurrency} />
                        <span className="budget-cap">{t.budget.balance}</span>
                      </span>
                      {editMode ? (
                        <span className="edit-mark" aria-hidden="true">
                          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M4 20h4L19 9l-4-4L4 16zM13 7l4 4" />
                          </svg>
                        </span>
                      ) : (
                        !c.hidden && (
                          <button type="button" className="plus" aria-label={`${t.categories.addExpense}: ${c.name}`} onClick={() => onAddTo(c.id)}>
                            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" aria-hidden="true">
                              <path d="M12 5v14M5 12h14" />
                            </svg>
                          </button>
                        )
                      )}
                    </div>
                    <div className={r.balance_minor < 0 ? 'budget-bar budget-bar-over' : 'budget-bar'} aria-hidden="true">
                      <span style={{ width: `${fillPercent(r.spent_minor, r.budgeted_minor)}%` }} />
                    </div>
                    <div className="budget-nums">
                      <button type="button" className="budget-set" disabled={editMode} onClick={() => setPlanning(c)}>
                        {t.budget.planned}
                        <Money minor={r.budgeted_minor} currency={baseCurrency} />
                      </button>
                      <span>
                        {t.budget.spent} <Money minor={r.spent_minor} currency={baseCurrency} />
                      </span>
                      {c.rollover && <span className="budget-roll">{t.budget.rolls}</span>}
                    </div>
                  </div>
                )
              })}
            </div>
          </section>
        )
      })}

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
