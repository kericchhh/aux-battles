import { useQuery } from '@tanstack/react-query'
import { Navigate, useLocation } from 'react-router'

import { meQuery } from '@/api/auth'
import { AppShell } from '@/components/app-shell'
import { PageLoader } from '@/components/page-loader'

export function RequireAuth() {
  const location = useLocation()
  const me = useQuery(meQuery)

  if (me.isPending) return <PageLoader />
  if (me.isError) throw me.error
  if (!me.data) return <Navigate to="/sign-in" replace state={{ from: location.pathname }} />
  return <AppShell user={me.data} />
}

export function GuestOnly({ children }: { children: React.ReactNode }) {
  const me = useQuery(meQuery)

  if (me.isPending) return <PageLoader />
  if (me.data) return <Navigate to="/" replace />
  return children
}
