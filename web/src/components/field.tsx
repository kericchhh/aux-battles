import { useId, type ComponentProps } from 'react'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type FieldProps = ComponentProps<typeof Input> & {
  label: string
  error?: string
  hint?: string
}

export function Field({ label, error, hint, ...props }: FieldProps) {
  const id = useId()
  const hintId = hint ? `${id}-hint` : undefined
  const errorId = error ? `${id}-error` : undefined

  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={[hintId, errorId].filter(Boolean).join(' ') || undefined}
        {...props}
      />
      {hint && (
        <p id={hintId} className="text-sm text-dim">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} className="animate-rise text-sm text-alarm">
          {error}
        </p>
      )}
    </div>
  )
}
