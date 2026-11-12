import { useQuery } from '@tanstack/react-query'
import { Navigate, useLocation } from 'react-router'

import { meQuery } from '@/api/auth'
import { returnPath } from '@/features/auth/return-path'
import { AppShell } from '@/components/layout/app-shell'
import { PageLoader } from '@/components/page-loader'
import { NotFound } from '@/routes/not-found'

export function RequireAuth() {
  const location = useLocation()
  const me = useQuery(meQuery)

  if (me.isPending) return <PageLoader />
  if (me.isError && me.data === undefined) throw me.error
  if (!me.data) return <Navigate to="/sign-in" replace state={{ from: location.pathname }} />
  return <AppShell user={me.data} />
}

export function GuestOnly({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const me = useQuery(meQuery)

  if (me.isPending) return <PageLoader />
  if (me.data) return <Navigate to={returnPath(location.state)} replace />
  return children
}

export function AdminOnly({ children }: { children: React.ReactNode }) {
  const me = useQuery(meQuery)

  if (me.data?.role !== 'ADMIN') return <NotFound />
  return children
}
