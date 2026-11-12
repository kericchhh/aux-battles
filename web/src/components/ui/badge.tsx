import type { ComponentProps } from 'react'

import { cn } from '@/lib/utils'

type BadgeProps = ComponentProps<'span'> & { tone?: 'idle' | 'live' | 'win' | 'alarm' }

export function Badge({ tone = 'idle', className, ...props }: BadgeProps) {
  return <span data-tone={tone} className={cn('badge', className)} {...props} />
}
