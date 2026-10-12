import { useState, type FormEvent } from 'react'
import { ApiError, createCategory, updateCategory, type Category, type CategoryGroup } from './api'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { ChipPicker } from './Pickers'
import { groupOrder } from './report'

type Props = {
  categories: Category[]
  onChanged: () => void
  onOpenCategory: (id: number) => void
  onAddTo: (id: number) => void
}

const groupOptions = groupOrder.map((g) => ({ value: g, label: t.groups[g] }))

export function CategoriesScreen({ categories, onChanged, onOpenCategory, onAddTo }: Props) {
  const [editMode, setEditMode] = useState(false)
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Category | null>(null)

  return (
    <main className="screen">
      <h1>{t.categories.title}</h1>

      <div className="toolbar">
        <button type="button" className={editMode ? 'chip chip-on' : 'chip'} onClick={() => setEditMode(!editMode)}>
          {editMode ? t.categories.editDone : t.categories.edit}
        </button>
        <button type="button" className="chip" onClick={() => setCreating(true)}>
          {t.categories.new}
        </button>
      </div>

      {groupOrder.map((g) => {
        const inGroup = categories.filter((c) => c.group === g)
        return (
          <section key={g}>
            <h2 className="day">{t.groups[g]}</h2>
            <div className="card list-card">
              {inGroup.length === 0 && <p className="hint tight pad">{t.categories.empty}</p>}
              {inGroup.map((c) => (
                <div key={c.id} className={c.hidden ? 'entry entry-muted' : 'entry'}>
                  <button type="button" className="entry-open" onClick={() => (editMode ? setEditing(c) : onOpenCategory(c.id))}>
                    <span className="entry-title">{c.name}</span>
                    <span className="entry-sub">{c.description}</span>
                  </button>
                  {c.hidden && <span className="badge-muted">{t.categories.hidden}</span>}
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
              ))}
            </div>
          </section>
        )
      })}

      {creating && (
        <Modal title={t.categories.new} onClose={() => setCreating(false)}>
          <CategoryForm
            initial={{ name: '', description: '', group: 'daily', hidden: false }}
            withGroup
            onSubmit={async (v) => {
              await createCategory(v.name, v.group)
              setCreating(false)
              onChanged()
            }}
          />
        </Modal>
      )}

      {editing !== null && (
        <Modal title={t.categories.editTitle} onClose={() => setEditing(null)}>
          <CategoryForm
            initial={{ name: editing.name, description: editing.description, group: editing.group, hidden: editing.hidden }}
            withDetails
            onSubmit={async (v) => {
              const patch: { name?: string; description?: string; hidden?: boolean } = {}
              if (v.name !== editing.name) patch.name = v.name
              if (v.description !== editing.description) patch.description = v.description
              if (v.hidden !== editing.hidden) patch.hidden = v.hidden
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

type Values = {
  name: string
  description: string
  group: CategoryGroup
  hidden: boolean
}

type FormProps = {
  initial: Values
  // withGroup: выбор группы при создании. withDetails: описание и «скрыта» при правке.
  withGroup?: boolean
  withDetails?: boolean
  onSubmit: (v: Values) => Promise<void>
}

function CategoryForm({ initial, withGroup = false, withDetails = false, onSubmit }: FormProps) {
  const [v, setV] = useState(initial)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await onSubmit(v)
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  return (
    <form className="form" onSubmit={submit}>
      <label className="field">
        <span className="label">{t.categories.name}</span>
        <input value={v.name} maxLength={40} autoFocus onChange={(e) => setV({ ...v, name: e.target.value })} />
      </label>
      {withGroup && <ChipPicker label={t.categories.group} value={v.group} options={groupOptions} onChange={(group) => setV({ ...v, group })} />}
      {withDetails && (
        <>
          <label className="field">
            <span className="label">{t.categories.description}</span>
            <input value={v.description} maxLength={200} onChange={(e) => setV({ ...v, description: e.target.value })} />
          </label>
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
