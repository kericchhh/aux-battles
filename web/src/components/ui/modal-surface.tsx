import type { ComponentProps, ReactNode } from 'react'

import { cn } from '@/lib/utils'

export function ModalSurface({
  className,
  children,
  close,
  ...props
}: ComponentProps<'div'> & { close?: ReactNode }) {
  return (
    <div className={cn('dialog', className)} {...props}>
      <div className="dialog-body">{children}</div>
      {close}
    </div>
  )
}
