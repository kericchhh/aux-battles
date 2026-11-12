import type { ReactNode } from 'react'

import { Brand } from './brand'

type AuthLayoutProps = {
  title: string
  description: string
  children: ReactNode
}

export function AuthLayout({ title, description, children }: AuthLayoutProps) {
  return (
    <main id="main" className="grid min-h-dvh place-items-center px-4 py-12">
      <div className="card-pop w-full max-w-sm">
        <Brand />
        <div className="grid gap-1">
          <h1 className="text-2xl">{title}</h1>
          <p className="text-dim">{description}</p>
        </div>
        {children}
      </div>
    </main>
  )
}
