import type { ReactNode } from 'react'

type AuthLayoutProps = {
  title: string
  description: string
  children: ReactNode
}

export function AuthLayout({ title, description, children }: AuthLayoutProps) {
  return (
    <main id="main" className="grid min-h-dvh place-items-center px-4 py-12">
      <div className="card-pop w-full max-w-sm">
        <div className="page-header">
          <h1 className="text-2xl">{title}</h1>
          <p>{description}</p>
        </div>
        {children}
      </div>
    </main>
  )
}
