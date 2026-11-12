import { QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen, waitFor } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { createQueryClient } from './query-client'
import { acceptSession, meQuery } from './auth'
import { ApiError } from './client'
import { battlesQuery } from './battles'
import { Home } from '@/routes/home'
import { AdminOnly, RequireAuth } from '@/routes/require-auth'
import { json, mockApi, mockFetch } from '@/test/render'

const user = { id: 'u1', username: 'ana', email: 'ana@test', role: 'USER' }

it('redirects home to sign in when a protected query reports session expiry', async () => {
  mockApi({
    'GET /battles': () => json({ message: 'expired' }, 401),
    'GET /users/me/stats': () => json({}),
  })
  const client = createQueryClient()
  client.setQueryData(meQuery.queryKey, user)
  const router = createMemoryRouter(
    [
      { element: <RequireAuth />, children: [{ index: true, element: <Home /> }] },
      { path: '/sign-in', element: <h1>Sign in page</h1> },
    ],
    { initialEntries: ['/'] },
  )
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  expect(await screen.findByText('Sign in page')).toBeInTheDocument()
  expect(router.state.location.state).toEqual({ from: '/' })
  expect(client.getQueryData(meQuery.queryKey)).toBeNull()
  client.clear()
})

it('does not grant admin access to users', async () => {
  mockApi({
    'GET /battles': () => json({ message: 'down' }, 500),
    'GET /users/me/stats': () => json({}),
  })
  const client = createQueryClient()
  client.setQueryData(meQuery.queryKey, user)
  const router = createMemoryRouter([
    {
      element: <RequireAuth />,
      children: [
        {
          index: true,
          element: (
            <AdminOnly>
              <h1>Upload secret</h1>
            </AdminOnly>
          ),
        },
      ],
    },
  ])
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  expect(await screen.findByRole('heading', { name: 'Dead air' })).toBeInTheDocument()
  expect(client.getQueryData(meQuery.queryKey)).toEqual(user)
  client.clear()
})

it('keeps cached identity when protected requests fail with a server error', async () => {
  mockApi({ 'GET /battles': () => json({ message: 'down' }, 500) })
  const client = createQueryClient()
  client.setQueryData(meQuery.queryKey, user)
  await expect(client.fetchQuery(battlesQuery)).rejects.toThrow('down')
  expect(client.getQueryData(meQuery.queryKey)).toEqual(user)
  client.clear()
})

it('cancels an older identity request before accepting a successful login', async () => {
  let finish: ((response: Response) => void) | undefined
  let calls = 0
  mockFetch(() =>
    ++calls === 1
      ? new Promise<Response>((resolve) => {
          finish = resolve
        })
      : json({ ...user, role: 'ADMIN' }),
  )
  const client = createQueryClient()
  const stale = client.fetchQuery(meQuery).catch(() => null)
  await waitFor(() => expect(finish).toBeDefined())
  await acceptSession(client)
  finish!(json({ message: 'old anonymous request' }, 401))
  await stale
  expect(client.getQueryData(meQuery.queryKey)).toEqual({ ...user, role: 'ADMIN' })
  client.clear()
})

it('keeps the authenticated app mounted when an identity refresh fails', async () => {
  mockApi({ 'GET /users/me': () => json({ message: 'temporary server error' }, 500) })
  const client = createQueryClient()
  client.setQueryData(meQuery.queryKey, user)
  const router = createMemoryRouter([
    {
      element: <RequireAuth />,
      errorElement: <p>App removed</p>,
      children: [{ index: true, element: <h1>Loaded app</h1> }],
    },
  ])
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  await act(async () => {
    await client.invalidateQueries({ queryKey: meQuery.queryKey })
  })
  expect(screen.getByText('Loaded app')).toBeInTheDocument()
  expect(screen.queryByText('App removed')).not.toBeInTheDocument()
  client.clear()
})

it('expires protected mutations but keeps rejected public credentials separate', async () => {
  const client = createQueryClient()
  const rejected = () => Promise.reject(new ApiError(401, 'unauthorized'))
  client.setQueryData(meQuery.queryKey, user)
  await expect(
    client
      .getMutationCache()
      .build(client, { mutationFn: rejected, meta: { requiresAuth: false } })
      .execute(undefined),
  ).rejects.toThrow('unauthorized')
  expect(client.getQueryData(meQuery.queryKey)).toEqual(user)
  await expect(
    client.getMutationCache().build(client, { mutationFn: rejected }).execute(undefined),
  ).rejects.toThrow('unauthorized')
  expect(client.getQueryData(meQuery.queryKey)).toBeNull()
  client.clear()
})
