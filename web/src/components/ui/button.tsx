import type { ComponentProps } from 'react'
import { Slot } from 'radix-ui'

import { cn } from '@/lib/utils'

type ButtonProps = ComponentProps<'button'> & {
  variant?: 'default' | 'secondary' | 'ghost' | 'destructive'
  size?: 'default' | 'icon'
  asChild?: boolean
}

export function Button({ className, variant, size, asChild, ...props }: ButtonProps) {
  const Comp = asChild ? Slot.Root : 'button'
  return (
    <Comp data-variant={variant} data-size={size} className={cn('btn', className)} {...props} />
  )
}
