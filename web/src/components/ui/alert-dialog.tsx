import type { ComponentProps } from 'react'
import { AlertDialog as Primitive } from 'radix-ui'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { ModalSurface } from './modal-surface'

export const AlertDialog = Primitive.Root
export const AlertDialogTrigger = Primitive.Trigger

export function AlertDialogContent({
  className,
  children,
  ...props
}: ComponentProps<typeof Primitive.Content>) {
  return (
    <Primitive.Portal>
      <Primitive.Overlay className="dialog-overlay" />
      <Primitive.Content asChild {...props}>
        <ModalSurface className={className}>{children}</ModalSurface>
      </Primitive.Content>
    </Primitive.Portal>
  )
}

export function AlertDialogTitle({ className, ...props }: ComponentProps<typeof Primitive.Title>) {
  return <Primitive.Title className={cn('text-2xl', className)} {...props} />
}

export function AlertDialogDescription({
  className,
  ...props
}: ComponentProps<typeof Primitive.Description>) {
  return <Primitive.Description className={cn('text-dim', className)} {...props} />
}

export function AlertDialogFooter({ className, ...props }: ComponentProps<'div'>) {
  return (
    <div
      className={cn('flex flex-col-reverse gap-3 sm:flex-row sm:justify-end', className)}
      {...props}
    />
  )
}

export function AlertDialogAction({
  variant,
  ...props
}: ComponentProps<typeof Primitive.Action> & Pick<ComponentProps<typeof Button>, 'variant'>) {
  return (
    <Button variant={variant} asChild>
      <Primitive.Action {...props} />
    </Button>
  )
}

export function AlertDialogCancel(props: ComponentProps<typeof Primitive.Cancel>) {
  return (
    <Button variant="secondary" asChild>
      <Primitive.Cancel {...props} />
    </Button>
  )
}
