import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { meQuery } from '@/api/auth'
import { AdminOnly, GuestOnly } from './require-auth'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { axeViolations, json, mockFetch, renderRoute } from '@/test/render'
import { SignIn } from './sign-in'

describe('SignIn', () => {
  it('has no detectable accessibility violations', async () => {
    const { container } = renderRoute(<SignIn />)
    expect(await axeViolations(container)).toEqual([])
  })

  it('links validation errors to their fields', async () => {
    renderRoute(<SignIn />)
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    const identifier = screen.getByLabelText('Username or email')
    expect(identifier).toHaveAttribute('aria-invalid', 'true')
    expect(identifier).toHaveAccessibleDescription('Enter your username or email')
  })

  it('announces rejected credentials', async () => {
    mockFetch(() => json({ message: 'invalid credentials' }, 401))
    renderRoute(<SignIn />)

    await userEvent.type(screen.getByLabelText('Username or email'), 'alice')
    await userEvent.type(screen.getByLabelText('Password'), 'wrong-password')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Wrong username, email or password.')
  })

  it('posts credentials to the proxied API and leaves the page', async () => {
    const fetch = mockFetch((url) =>
      url.endsWith('/users/login')
        ? json({ id: '1', username: 'alice', email: 'a@example.com' })
        : json({}),
    )
    const { router } = renderRoute(<SignIn />, '/sign-in')

    await userEvent.type(screen.getByLabelText('Username or email'), 'alice')
    await userEvent.type(screen.getByLabelText('Password'), 'correct-horse')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/'))
    const [url, init] = fetch.mock.calls[0]!
    expect(url).toBe('/api/users/login')
    expect(JSON.parse(String(init?.body))).toEqual({
      identifier: 'alice',
      password: 'correct-horse',
    })
  })
})

it('loads the canonical role and preserves a lazy protected return destination', async () => {
  let authenticated = false
  const user = { id: '1', username: 'admin', email: 'admin@test', role: 'ADMIN' }
  mockFetch((url) => {
    if (url.endsWith('/users/login')) {
      authenticated = true
      return json({ id: user.id, username: user.username, email: user.email })
    }
    return authenticated ? json(user) : json({ message: 'unauthorized' }, 401)
  })
  let loadUpload: (() => void) | undefined
  const uploaded = new Promise<void>((resolve) => {
    loadUpload = resolve
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createMemoryRouter(
    [
      {
        path: '/sign-in',
        element: (
          <GuestOnly>
            <SignIn />
          </GuestOnly>
        ),
      },
      { path: '/', element: <h1>Wrong home destination</h1> },
      {
        path: '/upload',
        lazy: async () => {
          await uploaded
          return {
            Component: () => (
              <AdminOnly>
                <h1>Admin upload page</h1>
              </AdminOnly>
            ),
          }
        },
      },
    ],
    { initialEntries: [{ pathname: '/sign-in', state: { from: '/upload' } }] },
  )
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  await userEvent.type(await screen.findByLabelText('Username or email'), 'admin')
  await userEvent.type(screen.getByLabelText('Password'), 'correct-horse')
  await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
  await waitFor(() => expect(client.getQueryData(meQuery.queryKey)).toEqual(user))
  await act(async () => {
    loadUpload!()
  })
  expect(await screen.findByText('Admin upload page')).toBeInTheDocument()
  expect(router.state.location.pathname).toBe('/upload')
  client.clear()
})
