import { useEffect, useState } from 'react'
import {
  addWallet,
  ApiError,
  commitImport,
  fetchCategories,
  fetchWallets,
  previewWallet,
  removeWallet,
  type Account,
  type Category,
  type ImportPreview,
  type Wallet,
} from './api'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'

type Props = {
  // Счета-кошельки: адрес привязывается к одному из них.
  wallets: Account[]
  onClose: () => void
  // Записи добавились: данные приложения нужно перечитать.
  onDone: () => void
}

const chains = ['eth', 'tron', 'sol'] as const

function shortAddress(a: string): string {
  return a.length > 14 ? `${a.slice(0, 6)}…${a.slice(-4)}` : a
}

// WalletSync: привязка публичного адреса к счёту-кошельку и подтягивание его переводов. Сначала превью, запись только
// по кнопке; уходящие переводы нужно отнести к категории.
export function WalletSync({ wallets, onClose, onDone }: Props) {
  const [accountId, setAccountId] = useState(wallets[0]?.id ?? 0)
  const [linked, setLinked] = useState<Wallet[]>([])
  const [chain, setChain] = useState<(typeof chains)[number]>('eth')
  const [address, setAddress] = useState('')
  const [consent, setConsent] = useState(false)
  const [categories, setCategories] = useState<Category[]>([])
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [picked, setPicked] = useState<Set<number>>(new Set())
  const [category, setCategory] = useState(0)
  const [saved, setSaved] = useState<number | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fail = (e: unknown) => setError(errorText(e instanceof ApiError ? e.code : 'internal'))

  useEffect(() => {
    fetchCategories()
      .then((list) => setCategories(list.filter((c) => !c.hidden)))
      .catch(() => setError(t.errors.internal))
  }, [])

  useEffect(() => {
    if (accountId === 0) return
    let alive = true
    fetchWallets(accountId)
      .then((list) => alive && setLinked(list))
      .catch(() => alive && setError(t.errors.internal))
    return () => {
      alive = false
    }
  }, [accountId])

  async function run(action: () => Promise<void>) {
    setBusy(true)
    setError(null)
    try {
      await action()
    } catch (e) {
      fail(e)
    } finally {
      setBusy(false)
    }
  }

  const link = () =>
    run(async () => {
      const w = await addWallet(accountId, chain, address.trim(), consent)
      setLinked((list) => (list.some((x) => x.id === w.id) ? list : [...list, w]))
      setAddress('')
    })

  const unlink = (id: number) =>
    run(async () => {
      await removeWallet(id)
      setLinked((list) => list.filter((w) => w.id !== id))
    })

  const read = (id: number) =>
    run(async () => {
      const p = await previewWallet(id)
      setPreview(p)
      // Новые переводы выбраны сразу, уже внесённые вносить нечего.
      setPicked(new Set(p.rows.flatMap((r, i) => (r.duplicate ? [] : [i]))))
    })

  const chosen = preview === null ? [] : preview.rows.filter((r, i) => picked.has(i) && !r.duplicate)
  const needsCategory = chosen.some((r) => r.amount_minor < 0)
  const ready = chosen.length > 0 && (!needsCategory || category !== 0)

  const save = () =>
    run(async () => {
      const rows = chosen.map((r) => ({ ...r, category_id: r.amount_minor < 0 ? category : null }))
      setSaved((await commitImport(accountId, rows)).saved)
    })

  function toggle(i: number) {
    setPicked((now) => {
      const next = new Set(now)
      if (!next.delete(i)) next.add(i)
      return next
    })
  }

  if (wallets.length === 0) {
    return (
      <Modal title={t.walletSync.title} onClose={onClose}>
        <p className="hint">{t.walletSync.noWallets}</p>
      </Modal>
    )
  }

  return (
    <Modal title={t.walletSync.title} onClose={saved === null ? onClose : onDone}>
      {saved !== null ? (
        <>
          <p>
            {t.walletSync.done}: {saved}
          </p>
          <button type="button" className="primary" onClick={onDone}>
            {t.close}
          </button>
        </>
      ) : preview === null ? (
        <>
          <p className="hint">{t.walletSync.hint}</p>
          <label className="field">
            <span className="label">{t.walletSync.account}</span>
            <select value={accountId} onChange={(e) => setAccountId(Number(e.target.value))}>
              {wallets.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </label>

          {linked.map((w) => (
            <div className="entry" key={w.id}>
              <span className="entry-open">
                <span className="entry-title">{t.walletSync.chains[w.chain as (typeof chains)[number]] ?? w.chain}</span>
                <span className="entry-sub">{shortAddress(w.address)}</span>
              </span>
              <button type="button" className="chip" disabled={busy} onClick={() => void read(w.id)}>
                {busy ? t.walletSync.reading : t.walletSync.read}
              </button>
              <button type="button" className="chip" disabled={busy} onClick={() => void unlink(w.id)}>
                {t.walletSync.unlink}
              </button>
            </div>
          ))}

          <h3>{t.walletSync.addTitle}</h3>
          <label className="field">
            <span className="label">{t.walletSync.chain}</span>
            <select value={chain} onChange={(e) => setChain(e.target.value as (typeof chains)[number])}>
              {chains.map((c) => (
                <option key={c} value={c}>
                  {t.walletSync.chains[c]}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span className="label">{t.walletSync.address}</span>
            <input value={address} onChange={(e) => setAddress(e.target.value)} autoCapitalize="off" autoCorrect="off" spellCheck={false} />
          </label>
          <label className="switch">
            <input type="checkbox" checked={consent} onChange={(e) => setConsent(e.target.checked)} />
            <span className="switch-track" aria-hidden="true" />
            <span>{t.walletSync.consent}</span>
          </label>
          {error !== null && <p className="error">{error}</p>}
          <button type="button" className="primary" disabled={busy || address.trim() === '' || !consent} onClick={() => void link()}>
            {t.walletSync.link}
          </button>
        </>
      ) : (
        <>
          {preview.rows.length === 0 && <p className="hint">{t.walletSync.empty}</p>}
          {preview.skipped > 0 && (
            <p className="hint tight">
              {t.walletSync.skipped}: {preview.skipped}
            </p>
          )}
          {preview.rows.map((r, i) => (
            <label className="entry" key={r.external_id}>
              <input type="checkbox" checked={picked.has(i) && !r.duplicate} disabled={r.duplicate} onChange={() => toggle(i)} />
              <span className="entry-open">
                <span className="entry-title">{r.date}</span>
                <span className="entry-sub">{r.duplicate ? t.walletSync.imported : r.description}</span>
              </span>
              <Money minor={r.amount_minor} currency={r.currency} signed />
            </label>
          ))}
          {needsCategory && (
            <label className="field">
              <span className="label">{t.walletSync.outCategory}</span>
              <select value={category} onChange={(e) => setCategory(Number(e.target.value))}>
                <option value={0}>{t.importFile.pick}</option>
                {categories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </label>
          )}
          {error !== null && <p className="error">{error}</p>}
          <button type="button" className="primary" disabled={busy || !ready} onClick={() => void save()}>
            {busy ? t.importFile.saving : `${t.walletSync.run} · ${chosen.length}`}
          </button>
          <button type="button" className="chip" onClick={() => setPreview(null)}>
            {t.back}
          </button>
        </>
      )}
    </Modal>
  )
}
