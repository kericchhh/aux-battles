import type { CSSProperties, ReactNode } from 'react'

type SegmentedProps<T> = {
  legend: string
  name: string
  options: readonly { value: T; label: string }[]
  value: T
  onChange: (value: T) => void
  hint?: ReactNode
}

export function Segmented<T>({ legend, name, options, value, onChange, hint }: SegmentedProps<T>) {
  const selected = options.findIndex((option) => option.value === value)

  return (
    <fieldset className="grid gap-2">
      <legend className="label mb-2">{legend}</legend>
      <div
        className="segmented"
        style={{ '--count': options.length, '--selected': selected } as CSSProperties}
      >
        <span className="segmented-thumb" aria-hidden />
        {options.map((option) => (
          <label key={option.label}>
            <input
              type="radio"
              name={name}
              value={option.label}
              checked={option.value === value}
              onChange={() => onChange(option.value)}
              className="sr-only"
            />
            <span>{option.label}</span>
          </label>
        ))}
      </div>
      {hint && <p className="text-dim">{hint}</p>}
    </fieldset>
  )
}
