import type { ComponentProps, CSSProperties } from 'react'
import { Tabs as Primitive } from 'radix-ui'

import { cn } from '@/lib/utils'

export const Tabs = Primitive.Root
export const TabsContent = Primitive.Content

type TabsListProps = ComponentProps<typeof Primitive.List> & {
  count: number
  selected: number
}

export function TabsList({ count, selected, className, children, ...props }: TabsListProps) {
  return (
    <Primitive.List
      className={cn('tabs-list', className)}
      style={{ '--count': count, '--selected': selected } as CSSProperties}
      {...props}
    >
      <span className="tabs-thumb" aria-hidden />
      {children}
    </Primitive.List>
  )
}

export function TabsTrigger({ className, ...props }: ComponentProps<typeof Primitive.Trigger>) {
  return <Primitive.Trigger className={cn('tabs-trigger', className)} {...props} />
}
