import { t } from './i18n'

export type Tab = 'operations' | 'categories' | 'accounts'

type Props = {
  tab: Tab
  onTab: (tab: Tab) => void
  onAdd: () => void
}

const tabs: { id: Tab; label: string; icon: string }[] = [
  { id: 'operations', label: t.tabs.operations, icon: 'M3 6h18M3 12h18M3 18h18' },
  { id: 'categories', label: t.tabs.categories, icon: 'M4 5h7v7H4zM13 5h7v7h-7zM4 14h7v6H4zM13 14h7v6h-7z' },
]

const accountsTab: { id: Tab; label: string; icon: string } = {
  id: 'accounts',
  label: t.tabs.accounts,
  icon: 'M3 7a3 3 0 013-3h12a3 3 0 013 3v10a3 3 0 01-3 3H6a3 3 0 01-3-3zM3 10h18M7 15h4',
}

function TabButton({ id, label, icon, active, onTab }: { id: Tab; label: string; icon: string; active: boolean; onTab: (tab: Tab) => void }) {
  return (
    <button type="button" className={active ? 'tab tab-active' : 'tab'} onClick={() => onTab(id)}>
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d={icon} />
      </svg>
      {label}
    </button>
  )
}

export function TabBar({ tab, onTab, onAdd }: Props) {
  return (
    <nav className="tabbar">
      {tabs.map((x) => (
        <TabButton key={x.id} {...x} active={tab === x.id} onTab={onTab} />
      ))}
      <button type="button" className="tab-add" aria-label={t.tabs.add} onClick={onAdd}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" aria-hidden="true">
          <path d="M12 5v14M5 12h14" />
        </svg>
      </button>
      <TabButton {...accountsTab} active={tab === accountsTab.id} onTab={onTab} />
    </nav>
  )
}
