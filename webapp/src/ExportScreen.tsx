import { useState } from 'react'
import { ApiError, fetchExport } from './api'
import { DateField } from './DateField'
import { errorText, t } from './i18n'
import { dayIn } from './ledger'
import { Modal } from './Modal'

// ExportScreen: выгрузка записей за период в CSV.
export function ExportScreen({ onClose }: { onClose: () => void }) {
  const today = dayIn(Intl.DateTimeFormat().resolvedOptions().timeZone)
  const [from, setFrom] = useState(`${today.slice(0, 8)}01`)
  const [to, setTo] = useState(today)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function download() {
    setBusy(true)
    setError(null)
    try {
      const url = URL.createObjectURL(await fetchExport(from, to))
      const link = document.createElement('a')
      link.href = url
      link.download = `moneykeeper-${from}_${to}.csv`
      link.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      setError(errorText(e instanceof ApiError ? e.code : 'internal'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={t.exportFile.title} onClose={onClose}>
      <p className="hint">{t.exportFile.hint}</p>
      <DateField label={t.exportFile.from} value={from} onChange={setFrom} placeholder={t.exportFile.from} />
      <DateField label={t.exportFile.to} value={to} onChange={setTo} placeholder={t.exportFile.to} />
      {error !== null && <p className="error">{error}</p>}
      <button type="button" className="primary" disabled={busy || from === '' || to === '' || to < from} onClick={() => void download()}>
        {t.exportFile.download}
      </button>
    </Modal>
  )
}
