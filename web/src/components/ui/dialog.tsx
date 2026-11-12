import type { ComponentProps } from 'react'
import { X } from 'lucide-react'
import { Dialog as Primitive } from 'radix-ui'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { ModalSurface } from './modal-surface'

export const Dialog = Primitive.Root
export const DialogTrigger = Primitive.Trigger

export function DialogContent({
  className,
  children,
  ...props
}: ComponentProps<typeof Primitive.Content>) {
  return (
    <Primitive.Portal>
      <Primitive.Overlay className="dialog-overlay" />
      <Primitive.Content asChild {...props}>
        <ModalSurface
          className={className}
          close={
            <Primitive.Close asChild>
              <Button variant="ghost" size="icon" aria-label="Close" className="dialog-close">
                <X aria-hidden />
              </Button>
            </Primitive.Close>
          }
        >
          {children}
        </ModalSurface>
      </Primitive.Content>
    </Primitive.Portal>
  )
}

export function DialogTitle({ className, ...props }: ComponentProps<typeof Primitive.Title>) {
  return <Primitive.Title className={cn('text-2xl', className)} {...props} />
}

export function DialogDescription({
  className,
  ...props
}: ComponentProps<typeof Primitive.Description>) {
  return <Primitive.Description className={cn('text-dim', className)} {...props} />
}
