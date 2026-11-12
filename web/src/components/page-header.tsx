import type { ReactNode } from 'react'

type PageHeaderProps = {
  title: ReactNode
  description?: ReactNode
  badge?: ReactNode
  action?: ReactNode
}

export function PageHeader({ title, description, badge, action }: PageHeaderProps) {
  return (
    <div className="page-header">
      <div className="grid gap-1">
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="page-title">{title}</h1>
          {badge}
        </div>
        {description && <p className="text-dim">{description}</p>}
      </div>
      {action}
    </div>
  )
}
