import { createBrowserRouter, Outlet } from 'react-router'
import { PageLoader } from '@/components/page-loader'
import { ErrorPage } from '@/routes/error-page'
import { NotFound } from '@/routes/not-found'
import { AdminOnly, GuestOnly, RequireAuth } from '@/routes/require-auth'

export const router = createBrowserRouter([
  {
    errorElement: <ErrorPage />,
    HydrateFallback: PageLoader,
    children: [
      {
        element: (
          <GuestOnly>
            <Outlet />
          </GuestOnly>
        ),
        children: [
          {
            path: '/sign-in',
            lazy: async () => ({ Component: (await import('@/routes/sign-in')).SignIn }),
          },
          {
            path: '/register',
            lazy: async () => ({ Component: (await import('@/routes/register')).Register }),
          },
        ],
      },
      {
        element: <RequireAuth />,
        children: [
          { index: true, lazy: async () => ({ Component: (await import('@/routes/home')).Home }) },
          {
            path: 'battles/:battleId',
            lazy: async () => ({ Component: (await import('@/routes/battle')).BattlePage }),
          },
          {
            element: (
              <AdminOnly>
                <Outlet />
              </AdminOnly>
            ),
            children: [
              {
                path: 'upload',
                lazy: async () => ({ Component: (await import('@/routes/upload')).Upload }),
              },
            ],
          },
          { path: '*', element: <NotFound /> },
        ],
      },
    ],
  },
])
