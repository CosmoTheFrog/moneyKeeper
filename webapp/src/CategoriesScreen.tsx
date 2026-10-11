import { useState, type FormEvent } from 'react'
import { ApiError, createCategory, updateCategory, type Category, type CategoryGroup } from './api'
import { errorText, t } from './i18n'

type Props = {
  categories: Category[]
  onChanged: () => void
}

const groupOrder: CategoryGroup[] = ['critical', 'debt', 'daily', 'rainy', 'goal']

export function CategoriesScreen({ categories, onChanged }: Props) {
  const [openId, setOpenId] = useState<number | null>(null)
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function run(action: () => Promise<unknown>): Promise<boolean> {
    setError(null)
    try {
      await action()
      onChanged()
      return true
    } catch (e) {
      setError(errorText(e instanceof ApiError ? e.code : 'internal'))
      return false
    }
  }

  return (
    <main className="screen">
      <header className="row">
        <h1>{t.categories.title}</h1>
        <button type="button" className="chip" onClick={() => setAdding(!adding)}>
          {t.categories.add}
        </button>
      </header>

      {error !== null && <p className="error">{error}</p>}

      {adding && (
        <NewCategory
          onCreate={async (name, group) => {
            if (await run(() => createCategory(name, group))) setAdding(false)
          }}
        />
      )}

      {groupOrder.map((g) => {
        const inGroup = categories.filter((c) => c.group === g)
        return (
          <section key={g}>
            <h2 className="day">{t.groups[g]}</h2>
            <div className="card list-card">
              {inGroup.length === 0 && <p className="hint tight pad">{t.categories.empty}</p>}
              {inGroup.map((c) => (
                <div key={c.id}>
                  <button type="button" className={c.hidden ? 'entry entry-muted' : 'entry'} onClick={() => setOpenId(openId === c.id ? null : c.id)}>
                    <span className="entry-main">
                      <span className="entry-title">{c.name}</span>
                      <span className="entry-sub">{c.description}</span>
                    </span>
                    {c.hidden && <span className="badge-muted">{t.categories.hidden}</span>}
                  </button>
                  {openId === c.id && <EditCategory category={c} onSave={(patch) => run(() => updateCategory(c.id, patch))} />}
                </div>
              ))}
            </div>
          </section>
        )
      })}
    </main>
  )
}

function NewCategory({ onCreate }: { onCreate: (name: string, group: CategoryGroup) => Promise<void> }) {
  const [name, setName] = useState('')
  const [group, setGroup] = useState<CategoryGroup>('daily')

  function submit(e: FormEvent) {
    e.preventDefault()
    void onCreate(name, group)
  }

  return (
    <form className="card" onSubmit={submit}>
      <label>
        {t.categories.name}
        <input value={name} maxLength={40} onChange={(e) => setName(e.target.value)} />
      </label>
      <label>
        {t.categories.group}
        <select value={group} onChange={(e) => setGroup(groupOrder.find((g) => g === e.target.value) ?? 'daily')}>
          {groupOrder.map((g) => (
            <option key={g} value={g}>
              {t.groups[g]}
            </option>
          ))}
        </select>
      </label>
      <button type="submit" className="primary" disabled={name.trim() === ''}>
        {t.save}
      </button>
    </form>
  )
}

type Patch = { name?: string; description?: string; hidden?: boolean }

function EditCategory({ category, onSave }: { category: Category; onSave: (patch: Patch) => Promise<boolean> }) {
  const [name, setName] = useState(category.name)
  const [description, setDescription] = useState(category.description)
  const [hidden, setHidden] = useState(category.hidden)

  function submit(e: FormEvent) {
    e.preventDefault()
    const patch: Patch = {}
    if (name !== category.name) patch.name = name
    if (description !== category.description) patch.description = description
    if (hidden !== category.hidden) patch.hidden = hidden
    if (Object.keys(patch).length > 0) void onSave(patch)
  }

  return (
    <form className="inline-form" onSubmit={submit}>
      <label>
        {t.categories.name}
        <input value={name} maxLength={40} onChange={(e) => setName(e.target.value)} />
      </label>
      <label>
        {t.categories.description}
        <input value={description} maxLength={200} onChange={(e) => setDescription(e.target.value)} />
      </label>
      <label className="check">
        <input type="checkbox" checked={hidden} onChange={(e) => setHidden(e.target.checked)} />
        {t.categories.hidden}
      </label>
      <button type="submit" className="primary" disabled={name.trim() === ''}>
        {t.save}
      </button>
    </form>
  )
}
