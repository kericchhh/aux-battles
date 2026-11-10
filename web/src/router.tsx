import { createBrowserRouter } from 'react-router'

import { ErrorPage } from '@/routes/error-page'
import { Home } from '@/routes/home'
import { NotFound } from '@/routes/not-found'
import { Register } from '@/routes/register'
import { GuestOnly, RequireAuth } from '@/routes/require-auth'
import { SignIn } from '@/routes/sign-in'

export const router = createBrowserRouter([
  {
    errorElement: <ErrorPage />,
    children: [
      {
        path: '/sign-in',
        element: (
          <GuestOnly>
            <SignIn />
          </GuestOnly>
        ),
      },
      {
        path: '/register',
        element: (
          <GuestOnly>
            <Register />
          </GuestOnly>
        ),
      },
      {
        element: <RequireAuth />,
        children: [
          { index: true, element: <Home /> },
          { path: '*', element: <NotFound /> },
        ],
      },
    ],
  },
])
