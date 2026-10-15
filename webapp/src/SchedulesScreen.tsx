import { useEffect, useState, type FormEvent } from 'react'
import {
  ApiError,
  createSchedule,
  deleteSchedule,
  fetchOccurrences,
  skipOccurrence,
  updateSchedule,
  type Account,
  type AmountKind,
  type Category,
  type CategoryGroup,
  type Occurrence,
  type Schedule,
  type Transaction,
  type WeekendShift,
} from './api'
import { formatRange } from './calendar'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'
import { formatInput, parseAmount } from './money'
import { ChipPicker, CurrencyPicker, Segmented, SheetPicker } from './Pickers'
import { daysText, describeRepeat, groupOccurrences, parseDays, type Section } from './schedule'

type Props = {
  groups: CategoryGroup[]
  categories: Category[]
  accounts: Account[]
  baseCurrency: string
  currencies: string[]
  today: string
  // Записи бюджета: после новой траты статусы платежей пересчитываются.
  ledger: Transaction[]
  onBack: () => void
  onAddExpense: (categoryId: number) => void
}

const sections: Section[] = ['overdue', 'soon', 'later', 'closed']

function Amount({ s }: { s: Schedule }) {
  if (s.amount_kind === 'range' && s.amount_max_minor !== null) {
    return (
      <span className="sched-amount">
        <Money minor={s.amount_minor} currency={s.currency} /> – <Money minor={s.amount_max_minor} currency={s.currency} />
      </span>
    )
  }
  return (
    <span className="sched-amount">
      {s.amount_kind === 'approx' && `${t.schedules.approx} `}
      <Money minor={s.amount_minor} currency={s.currency} />
    </span>
  )
}

// Платежи: что и когда нужно оплатить. Оплаченным платёж становится сам, когда найдётся подходящая трата.
export function SchedulesScreen({ groups, categories, accounts, baseCurrency, currencies, today, ledger, onBack, onAddExpense }: Props) {
  const [list, setList] = useState<Occurrence[] | null>(null)
  const [failed, setFailed] = useState(false)
  const [version, setVersion] = useState(0)
  const [form, setForm] = useState<Schedule | 'new' | null>(null)
  const [picked, setPicked] = useState<Occurrence | null>(null)

  useEffect(() => {
    let alive = true
    fetchOccurrences()
      .then((l) => alive && setList(l))
      .catch(() => alive && setFailed(true))
    return () => {
      alive = false
    }
  }, [version, ledger])

  const refresh = () => setVersion((v) => v + 1)
  const grouped = groupOccurrences(list ?? [], today)
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))

  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onBack}>
          {t.back}
        </button>
        <button type="button" className="chip" onClick={() => setForm('new')}>
          {t.schedules.add}
        </button>
      </header>
      <h1>{t.schedules.title}</h1>

      {failed && <p className="error">{t.errors.internal}</p>}
      {list !== null && list.length === 0 && <p className="hint">{t.schedules.empty}</p>}

      {sections.map(
        (section) =>
          grouped[section].length > 0 && (
            <section key={section}>
              <h2 className="day">{t.schedules.sections[section]}</h2>
              <div className="card list-card">
                {grouped[section].map((o) => (
                  <div key={`${o.schedule.id}-${o.due_on}`} className={section === 'closed' ? 'entry entry-muted' : 'entry'}>
                    <button type="button" className="entry-open" onClick={() => setPicked(o)}>
                      <span className="entry-title">{o.schedule.name}</span>
                      <span className="entry-sub">
                        {formatRange({ from: o.due_on, to: o.due_on }, today)} · {categoryName.get(o.schedule.category_id) ?? ''}
                      </span>
                    </button>
                    <span className="sched-right">
                      <Amount s={o.schedule} />
                      <span className={`sched-status sched-${o.status}`}>{t.schedules.status[o.status]}</span>
                    </span>
                  </div>
                ))}
              </div>
            </section>
          ),
      )}

      {picked !== null && (
        <Modal title={picked.schedule.name} onClose={() => setPicked(null)}>
          <Actions
            occurrence={picked}
            today={today}
            onEnter={() => {
              setPicked(null)
              onAddExpense(picked.schedule.category_id)
            }}
            onEdit={() => {
              setPicked(null)
              setForm(picked.schedule)
            }}
            onChanged={() => {
              setPicked(null)
              refresh()
            }}
          />
        </Modal>
      )}

      {form !== null && (
        <Modal title={form === 'new' ? t.schedules.add : t.schedules.editTitle} onClose={() => setForm(null)}>
          <ScheduleForm
            schedule={form === 'new' ? null : form}
            groups={groups}
            categories={categories}
            accounts={accounts}
            baseCurrency={baseCurrency}
            currencies={currencies}
            onDone={() => {
              setForm(null)
              refresh()
            }}
          />
        </Modal>
      )}
    </main>
  )
}

type ActionsProps = {
  occurrence: Occurrence
  today: string
  onEnter: () => void
  onEdit: () => void
  onChanged: () => void
}

function Actions({ occurrence, today, onEnter, onEdit, onChanged }: ActionsProps) {
  const { schedule: s } = occurrence
  const [confirm, setConfirm] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const open = occurrence.status === 'upcoming' || occurrence.status === 'overdue'

  async function run(action: () => Promise<void>) {
    try {
      await action()
      onChanged()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
    }
  }

  return (
    <div className="stack">
      <div className="card">
        <p className="hint tight">
          {formatRange({ from: occurrence.due_on, to: occurrence.due_on }, today)} · {t.schedules.status[occurrence.status]}
        </p>
        <Amount s={s} />
        <p className="hint tight">{describeRepeat(s.every_months, s.days, t.schedules.repeatEvery)}</p>
        <p className="hint tight">{t.schedules.matchHint}</p>
      </div>
      {open && (
        <button type="button" className="primary" onClick={onEnter}>
          {t.schedules.enter}
        </button>
      )}
      {open && (
        <button type="button" className="chip" onClick={() => void run(() => skipOccurrence(s.id, occurrence.due_on))}>
          {t.schedules.skip}
        </button>
      )}
      <button type="button" className="chip" onClick={onEdit}>
        {t.schedules.change}
      </button>
      <button type="button" className={confirm ? 'danger danger-on' : 'danger'} onClick={() => (confirm ? void run(() => deleteSchedule(s.id)) : setConfirm(true))}>
        {confirm ? t.schedules.confirmRemove : t.schedules.remove}
      </button>
      {error !== null && <p className="error">{error}</p>}
    </div>
  )
}

type FormProps = {
  schedule: Schedule | null
  groups: CategoryGroup[]
  categories: Category[]
  accounts: Account[]
  baseCurrency: string
  currencies: string[]
  onDone: () => void
}

const kindOptions = (['exact', 'approx', 'range'] as const).map((k) => ({ value: k, label: t.schedules.kinds[k] }))
const everyOptions = ([1, 2, 3, 6, 12] as const).map((n) => ({ value: n, label: t.schedules.every[n] }))
const shiftOptions = (['none', 'back', 'forward'] as const).map((s) => ({ value: s, label: t.schedules.shifts[s] }))
const remindOptions = ([0, 1, 2, 3, 7] as const).map((n) => ({ value: n, label: t.schedules.reminds[n] }))

function ScheduleForm({ schedule, groups, categories, accounts, baseCurrency, currencies, onDone }: FormProps) {
  const [name, setName] = useState(schedule?.name ?? '')
  const [categoryId, setCategoryId] = useState(schedule?.category_id ?? 0)
  const [accountId, setAccountId] = useState(schedule?.account_id ?? 0)
  const [currency, setCurrency] = useState(schedule?.currency ?? baseCurrency)
  const [kind, setKind] = useState<AmountKind>(schedule?.amount_kind ?? 'exact')
  const [amount, setAmount] = useState(schedule === null ? '' : formatInput(schedule.amount_minor, schedule.currency))
  const [amountMax, setAmountMax] = useState(schedule?.amount_max_minor == null ? '' : formatInput(schedule.amount_max_minor, schedule.currency))
  const [every, setEvery] = useState(schedule?.every_months ?? 1)
  const [days, setDays] = useState(schedule === null ? '' : daysText(schedule.days))
  const [lastDay, setLastDay] = useState(schedule?.days.includes(0) ?? false)
  const [shift, setShift] = useState<WeekendShift>(schedule?.weekend_shift ?? 'none')
  const [remind, setRemind] = useState(schedule?.remind_before ?? 1)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const categoryGroups = groups
    .map((g) => ({
      title: g.name,
      options: categories.filter((c) => c.group_id === g.id && (!c.hidden || c.id === schedule?.category_id)).map((c) => ({ value: c.id, label: c.name })),
    }))
    .filter((g) => g.options.length > 0)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const minor = parseAmount(amount, currency)
    const maxMinor = kind === 'range' ? parseAmount(amountMax, currency) : null
    const parsedDays = parseDays(days, lastDay)
    if (minor === null || (kind === 'range' && maxMinor === null)) return setError(errorText('invalid_amount'))
    if (parsedDays === null) return setError(t.schedules.daysError)
    if (categoryId === 0) return setError(t.add.pickCategory)

    setBusy(true)
    try {
      const input = {
        name,
        category_id: categoryId,
        account_id: accountId === 0 ? null : accountId,
        currency,
        amount_kind: kind,
        amount_minor: minor,
        amount_max_minor: maxMinor,
        every_months: every,
        days: parsedDays,
        weekend_shift: shift,
        remind_before: remind,
      }
      if (schedule === null) await createSchedule(input)
      else await updateSchedule(schedule.id, input)
      onDone()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  return (
    <form className="form" onSubmit={submit}>
      <label className="field">
        <span className="label">{t.schedules.name}</span>
        <input value={name} maxLength={40} placeholder={t.schedules.namePlaceholder} autoFocus={schedule === null} onChange={(e) => setName(e.target.value)} />
      </label>
      <SheetPicker label={t.schedules.category} value={categoryId === 0 ? null : categoryId} placeholder={t.add.pickCategory} groups={categoryGroups} onChange={setCategoryId} />

      <span className="label">{t.schedules.amountKind}</span>
      <Segmented value={kind} options={kindOptions} onChange={setKind} />
      <CurrencyPicker label={t.add.currency} value={currency} options={currencies} onChange={setCurrency} />
      <label className="field">
        <span className="label">{t.schedules.amount}</span>
        <input inputMode="decimal" value={amount} placeholder="0,00" onChange={(e) => setAmount(e.target.value)} />
      </label>
      {kind === 'range' && (
        <label className="field">
          <span className="label">{t.schedules.amountTo}</span>
          <input inputMode="decimal" value={amountMax} placeholder="0,00" onChange={(e) => setAmountMax(e.target.value)} />
        </label>
      )}
      {kind === 'approx' && <p className="hint tight">{t.schedules.approxHint}</p>}

      <ChipPicker label={t.schedules.repeat} value={every} options={everyOptions} onChange={setEvery} />
      <label className="field">
        <span className="label">{t.schedules.days}</span>
        <input value={days} placeholder={t.schedules.daysPlaceholder} onChange={(e) => setDays(e.target.value)} />
      </label>
      <label className="switch">
        <input type="checkbox" checked={lastDay} onChange={(e) => setLastDay(e.target.checked)} />
        <span className="switch-track" aria-hidden="true" />
        <span>{t.schedules.lastDay}</span>
      </label>
      <ChipPicker label={t.schedules.shift} value={shift} options={shiftOptions} onChange={setShift} />

      <div className="field">
        <span className="label">{t.schedules.account}</span>
        <div className="choices">
          <button type="button" className={accountId === 0 ? 'choice choice-on' : 'choice'} onClick={() => setAccountId(0)}>
            <span>{t.schedules.anyAccount}</span>
          </button>
          {accounts
            .filter((a) => !a.off_budget)
            .map((a) => (
              <button key={a.id} type="button" className={a.id === accountId ? 'choice choice-on' : 'choice'} onClick={() => setAccountId(a.id)}>
                <span>{a.name}</span>
                <span className="choice-meta">{a.currency}</span>
              </button>
            ))}
        </div>
      </div>

      <ChipPicker label={t.schedules.remind} value={remind} options={remindOptions} onChange={setRemind} />
      <p className="hint tight">{t.schedules.remindHint}</p>

      {error !== null && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy || name.trim() === ''}>
        {t.save}
      </button>
    </form>
  )
}
