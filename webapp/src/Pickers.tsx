import { useState } from 'react'
import { Modal } from './Modal'

export type Option<T> = {
  value: T
  label: string
}

type ChipProps<T> = {
  label?: string
  value: T
  options: Option<T>[]
  onChange: (value: T) => void
  disabled?: boolean
}

// ChipPicker показывает все варианты рядом, подходит для коротких списков: тип счёта, валюта, группа.
export function ChipPicker<T extends string | number>({ label, value, options, onChange, disabled = false }: ChipProps<T>) {
  return (
    <div className="field">
      {label !== undefined && <span className="label">{label}</span>}
      <div className="chips" role="radiogroup" aria-label={label}>
        {options.map((o) => (
          <button
            key={o.value}
            type="button"
            role="radio"
            aria-checked={o.value === value}
            disabled={disabled}
            className={o.value === value ? 'pill pill-on' : 'pill'}
            onClick={() => onChange(o.value)}
          >
            {o.label}
          </button>
        ))}
      </div>
    </div>
  )
}

export type OptionGroup<T> = {
  title: string
  options: Option<T>[]
}

type SheetProps<T> = {
  label: string
  value: T | null
  placeholder: string
  groups: OptionGroup<T>[]
  onChange: (value: T) => void
}

// SheetPicker открывает длинный список в нижнем окне, варианты разложены по группам.
export function SheetPicker<T extends string | number>({ label, value, placeholder, groups, onChange }: SheetProps<T>) {
  const [open, setOpen] = useState(false)
  const current = groups.flatMap((g) => g.options).find((o) => o.value === value)

  return (
    <div className="field">
      <span className="label">{label}</span>
      <button type="button" className="select" onClick={() => setOpen(true)}>
        <span className={current === undefined ? 'select-empty' : ''}>{current?.label ?? placeholder}</span>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M6 9l6 6 6-6" />
        </svg>
      </button>
      {open && (
        <Modal title={label} sheet onClose={() => setOpen(false)}>
          {groups.map((g) => (
            <section key={g.title}>
              <h3 className="sheet-group">{g.title}</h3>
              {g.options.map((o) => (
                <button
                  key={o.value}
                  type="button"
                  className={o.value === value ? 'sheet-row sheet-row-on' : 'sheet-row'}
                  onClick={() => {
                    onChange(o.value)
                    setOpen(false)
                  }}
                >
                  <span>{o.label}</span>
                  {o.value === value && (
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                      <path d="M5 12l5 5 9-10" />
                    </svg>
                  )}
                </button>
              ))}
            </section>
          ))}
        </Modal>
      )}
    </div>
  )
}
