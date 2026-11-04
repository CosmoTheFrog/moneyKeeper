import { useEffect, useState } from 'react'
import { ApiError, commitImport, fetchCategories, previewImportFile, type Account, type Category, type ImportPreview } from './api'
import { errorText, t } from './i18n'
import { Modal } from './Modal'

type Props = {
  accounts: Account[]
  onClose: () => void
  onDone: () => void
}

// ImportScreen: перенос трат и доходов из xlsx-отчёта прежнего бота. Сначала превью, запись только по кнопке.
export function ImportScreen({ accounts, onClose, onDone }: Props) {
  const [accountId, setAccountId] = useState(accounts[0]?.id ?? 0)
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  // Категория бюджета для каждого названия из файла, 0 значит ещё не выбрана.
  const [mapping, setMapping] = useState<Record<string, number>>({})
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchCategories()
      .then((list) => setCategories(list.filter((c) => !c.hidden)))
      .catch(() => setError(t.errors.internal))
  }, [])

  const fail = (e: unknown) => setError(errorText(e instanceof ApiError ? e.code : 'internal'))

  async function read() {
    if (file === null) return
    setBusy(true)
    setError(null)
    try {
      setPreview(await previewImportFile(accountId, file))
      setMapping({})
    } catch (e) {
      fail(e)
    } finally {
      setBusy(false)
    }
  }

  const fresh = preview?.rows.filter((r) => !r.duplicate) ?? []
  const unmapped = [...new Set(fresh.filter((r) => r.category_id === null && r.amount_minor < 0).map((r) => r.source_category))]
  const ready = fresh.length > 0 && unmapped.every((name) => (mapping[name] ?? 0) !== 0)

  async function save() {
    setBusy(true)
    setError(null)
    try {
      const rows = fresh.map((r) => ({ ...r, category_id: r.category_id ?? (mapping[r.source_category] || null) }))
      setSaved((await commitImport(accountId, rows)).saved)
    } catch (e) {
      fail(e)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={t.importFile.title} onClose={saved === null ? onClose : onDone}>
      {saved !== null ? (
        <>
          <p>
            {t.importFile.done}: {saved}
          </p>
          <button type="button" className="primary" onClick={onDone}>
            {t.close}
          </button>
        </>
      ) : preview === null ? (
        <>
          <p className="hint">{t.importFile.hint}</p>
          <label className="field">
            <span className="label">{t.importFile.account}</span>
            <select value={accountId} onChange={(e) => setAccountId(Number(e.target.value))}>
              {accounts.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span className="label">{t.importFile.file}</span>
            <input type="file" accept=".xlsx,.pdf" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
          </label>
          {error !== null && <p className="error">{error}</p>}
          <button type="button" className="primary" disabled={busy || file === null || accountId === 0} onClick={() => void read()}>
            {busy ? t.importFile.reading : t.importFile.read}
          </button>
        </>
      ) : (
        <>
          <p>
            {t.importFile.willImport}: {fresh.length}
          </p>
          {preview.rows.length > fresh.length && (
            <p className="hint tight">
              {t.importFile.duplicates}: {preview.rows.length - fresh.length}
            </p>
          )}
          {preview.skipped > 0 && (
            <p className="hint tight">
              {t.importFile.skipped}: {preview.skipped}
            </p>
          )}
          {unmapped.length > 0 && (
            <>
              <h3>{t.importFile.mapTitle}</h3>
              <p className="hint tight">{t.importFile.mapHint}</p>
              {unmapped.map((name) => (
                <label className="field" key={name}>
                  <span className="label">{name}</span>
                  <select value={mapping[name] ?? 0} onChange={(e) => setMapping({ ...mapping, [name]: Number(e.target.value) })}>
                    <option value={0}>{t.importFile.pick}</option>
                    {categories.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </label>
              ))}
            </>
          )}
          {error !== null && <p className="error">{error}</p>}
          <button type="button" className="primary" disabled={busy || !ready} onClick={() => void save()}>
            {busy ? t.importFile.saving : t.importFile.run}
          </button>
        </>
      )}
    </Modal>
  )
}
