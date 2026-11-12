import { useId, type ComponentProps, type ReactNode } from 'react'

import { cn } from '@/lib/utils'

type CardProps = Omit<ComponentProps<'section'>, 'title'> & {
  as?: 'section' | 'aside'
  title: ReactNode
  description?: ReactNode
  action?: ReactNode
}

export function Card({
  as: Tag = 'section',
  title,
  description,
  action,
  className,
  children,
  ...props
}: CardProps) {
  const id = useId()

  return (
    <Tag aria-labelledby={id} className={cn('card', className)} {...props}>
      <div className="card-header">
        <div className="grid gap-1">
          <h2 id={id} className="card-title">
            {title}
          </h2>
          {description && <p className="text-dim">{description}</p>}
        </div>
        {action}
      </div>
      {children}
    </Tag>
  )
}
