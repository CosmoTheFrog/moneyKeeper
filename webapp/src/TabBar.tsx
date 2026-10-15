import { t } from './i18n'

export type Tab = 'operations' | 'budget' | 'reports' | 'accounts'

type Props = {
  tab: Tab
  onTab: (tab: Tab) => void
  onAdd: () => void
}

type Item = { id: Tab; label: string; icon: string }

// Слева от «+» операции и бюджет, справа отчёты и счета.
const left: Item[] = [
  { id: 'operations', label: t.tabs.operations, icon: 'M3 6h18M3 12h18M3 18h18' },
  { id: 'budget', label: t.tabs.budget, icon: 'M4 5h7v7H4zM13 5h7v7h-7zM4 14h7v6H4zM13 14h7v6h-7z' },
]

const right: Item[] = [
  { id: 'reports', label: t.tabs.reports, icon: 'M5 20v-9M12 20V5M19 20v-6' },
  { id: 'accounts', label: t.tabs.accounts, icon: 'M3 7a3 3 0 013-3h12a3 3 0 013 3v10a3 3 0 01-3 3H6a3 3 0 01-3-3zM3 10h18M7 15h4' },
]

function TabButton({ item, active, onTab }: { item: Item; active: boolean; onTab: (tab: Tab) => void }) {
  return (
    <button type="button" className={active ? 'tab tab-active' : 'tab'} onClick={() => onTab(item.id)}>
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d={item.icon} />
      </svg>
      {item.label}
    </button>
  )
}

export function TabBar({ tab, onTab, onAdd }: Props) {
  return (
    <nav className="tabbar">
      {left.map((x) => (
        <TabButton key={x.id} item={x} active={tab === x.id} onTab={onTab} />
      ))}
      <button type="button" className="tab-add" aria-label={t.tabs.add} onClick={onAdd}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" aria-hidden="true">
          <path d="M12 5v14M5 12h14" />
        </svg>
      </button>
      {right.map((x) => (
        <TabButton key={x.id} item={x} active={tab === x.id} onTab={onTab} />
      ))}
    </nav>
  )
}
