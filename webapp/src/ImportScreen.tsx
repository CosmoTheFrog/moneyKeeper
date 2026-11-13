import { useEffect, useState } from 'react'
import { ApiError, commitImport, fetchCategories, previewImportFile, previewImportText, undoImport, type Account, type Category, type ImportPreview, type ImportRow } from './api'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'

type Props = {
  accounts: Account[]
  onClose: () => void
  onDone: () => void
}

const isText = (f: File) => /\.(csv|txt)$/i.test(f.name)

// Ключ «похожих» строк: по нему выбор категории для одной строки применяется к остальным с тем же продавцом или MCC.
const similarKey = (r: ImportRow) => (r.description.trim() !== '' ? `d:${r.description.trim().toLowerCase()}` : `s:${r.source_category}`)

// ImportScreen: перенос трат и доходов из xlsx-отчёта прежнего бота, PDF-выписки TBC или текста выписки (csv, txt).
// Сначала список строк, где каждую можно снять или поправить, запись только по кнопке; сразу после записи импорт
// можно отменить.
export function ImportScreen({ accounts, onClose, onDone }: Props) {
  const [accountId, setAccountId] = useState(accounts[0]?.id ?? 0)
  const [file, setFile] = useState<File | null>(null)
  const [consent, setConsent] = useState(false)
  // PDF не от TBC разбирает нейросеть: сервер просит согласие, и переключатель появляется после первой попытки.
  const [pdfNeedsConsent, setPdfNeedsConsent] = useState(false)
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  // Категория и отметка «записать» для каждой строки превью по её номеру; 0 значит категория ещё не выбрана.
  const [choice, setChoice] = useState<number[]>([])
  const [included, setIncluded] = useState<boolean[]>([])
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState<{ count: number; ids: number[] } | null>(null)
  const [undone, setUndone] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchCategories()
      .then((list) => setCategories(list.filter((c) => !c.hidden)))
      .catch(() => setError(t.errors.internal))
  }, [])

  const fail = (e: unknown) => setError(errorText(e instanceof ApiError ? e.code : 'internal'))
  const needsConsent = file !== null && (isText(file) || pdfNeedsConsent)

  async function read() {
    if (file === null) return
    setBusy(true)
    setError(null)
    try {
      const p = isText(file) ? await previewImportText(accountId, await file.text(), consent) : await previewImportFile(accountId, file, consent)
      setPreview(p)
      setChoice(p.rows.map((r) => r.category_id ?? 0))
      // Уже внесённые строки не отмечены, но их можно отметить: так выглядит «это другая покупка».
      setIncluded(p.rows.map((r) => !r.duplicate))
    } catch (e) {
      if (e instanceof ApiError && e.code === 'import_consent_required' && !isText(file)) {
        setPdfNeedsConsent(true)
        setError(t.importFile.pdfConsent)
      } else fail(e)
    } finally {
      setBusy(false)
    }
  }

  const rows = preview?.rows ?? []
  const chosen = rows.map((r, i) => ({ r, i })).filter(({ i }) => included[i])
  const ready = chosen.length > 0 && chosen.every(({ r, i }) => r.amount_minor >= 0 || (choice[i] ?? 0) !== 0)

  function pickCategory(index: number, id: number) {
    const row = rows[index]
    if (row === undefined) return
    const was = choice[index] ?? 0
    const key = similarKey(row)
    setChoice(choice.map((c, i) => (i === index || (rows[i] !== undefined && similarKey(rows[i]) === key && rows[i].amount_minor < 0 && (c === 0 || c === was)) ? id : c)))
  }

  async function save() {
    setBusy(true)
    setError(null)
    try {
      const picked = chosen.map(({ r, i }) => ({ ...r, category_id: r.amount_minor < 0 ? choice[i] || null : null }))
      const res = await commitImport(accountId, picked)
      setSaved({ count: res.saved, ids: res.ids })
    } catch (e) {
      fail(e)
    } finally {
      setBusy(false)
    }
  }

  async function undo() {
    if (saved === null) return
    setBusy(true)
    setError(null)
    try {
      await undoImport(saved.ids)
      setUndone(true)
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
          <p>{undone ? t.importFile.undone : `${t.importFile.done}: ${saved.count}`}</p>
          {error !== null && <p className="error">{error}</p>}
          {!undone && saved.ids.length > 0 && (
            <button type="button" className="danger" disabled={busy} onClick={() => void undo()}>
              {t.importFile.undo}
            </button>
          )}
          <button type="button" className="primary" onClick={onDone}>
            {t.close}
          </button>
        </>
      ) : preview === null ? (
        <>
          <p className="hint">{t.importFile.hint}</p>
          {accounts.length > 1 && (
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
          )}
          <label className="field">
            <span className="label">{t.importFile.file}</span>
            <input type="file" accept=".xlsx,.pdf,.csv,.txt" onChange={(e) => {
                setFile(e.target.files?.[0] ?? null)
                setPdfNeedsConsent(false)
              }} />
          </label>
          {needsConsent && (
            <label className="switch">
              <input type="checkbox" checked={consent} onChange={(e) => setConsent(e.target.checked)} />
              <span className="switch-track" aria-hidden="true" />
              <span>{t.importFile.consent}</span>
            </label>
          )}
          {error !== null && <p className="error">{error}</p>}
          <button type="button" className="primary" disabled={busy || file === null || accountId === 0 || (needsConsent && !consent)} onClick={() => void read()}>
            {busy ? t.importFile.reading : t.importFile.read}
          </button>
        </>
      ) : (
        <>
          <p>
            {t.importFile.willImport}: {chosen.length}
          </p>
          {preview.skipped > 0 && (
            <p className="hint tight">
              {t.importFile.skipped}: {preview.skipped}
            </p>
          )}
          {preview.other_currencies.length > 0 && <p className="error">{t.importFile.otherCurrency.replace('{codes}', preview.other_currencies.join(', '))}</p>}
          <p className="hint tight">{t.importFile.rowsHint}</p>
          <div className="imp-list">
            {rows.map((r, i) => (
              <div key={i} className={included[i] ? 'imp-row' : 'imp-row imp-off'}>
                <label className="imp-head">
                  <input type="checkbox" checked={included[i] ?? false} onChange={(e) => setIncluded(included.map((v, j) => (j === i ? e.target.checked : v)))} />
                  <span className="imp-desc">
                    {r.description === '' ? t.importFile.noDescription : r.description}
                    {r.duplicate && <span className="badge-muted">{t.importFile.duplicateTag}</span>}
                  </span>
                  <Money minor={r.amount_minor} currency={r.currency} signed />
                </label>
                <div className="imp-sub">
                  <span className="hint tight">{r.date}</span>
                  {r.amount_minor < 0 && (
                    <select aria-label={t.add.category} value={choice[i] ?? 0} onChange={(e) => pickCategory(i, Number(e.target.value))}>
                      <option value={0}>{r.source_category === '' ? t.importFile.pick : `${t.importFile.pick} (${r.source_category})`}</option>
                      {categories.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                    </select>
                  )}
                </div>
              </div>
            ))}
          </div>
          {error !== null && <p className="error">{error}</p>}
          <button type="button" className="primary" disabled={busy || !ready} onClick={() => void save()}>
            {busy ? t.importFile.saving : t.importFile.run}
          </button>
        </>
      )}
    </Modal>
  )
}
