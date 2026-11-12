import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen } from '@testing-library/react'
import { RouterProvider } from 'react-router'

import { json, mockFetch } from '@/test/render'
import { router } from './router'

const pending = vi.hoisted(() => {
  let resolve!: () => void
  const promise = new Promise<void>((done) => {
    resolve = done
  })
  return { promise, resolve }
})

vi.mock('@/routes/home', async () => {
  await pending.promise
  return { Home: () => <h1>Home route loaded</h1> }
})

it('shows the existing loader until the initial route module has loaded', async () => {
  mockFetch(() => json({ id: 'u1', username: 'ana', email: 'ana@test' }))
  const warning = vi.spyOn(console, 'warn')
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  expect(screen.getByRole('status')).toHaveTextContent('Loading…')
  expect(screen.queryByText('Home route loaded')).not.toBeInTheDocument()
  await act(async () => {
    pending.resolve()
  })
  expect(await screen.findByRole('heading', { name: 'Home route loaded' })).toBeInTheDocument()
  expect(
    warning.mock.calls.flat().some((message) => String(message).includes('HydrateFallback')),
  ).toBe(false)
  warning.mockRestore()
  client.clear()
  router.dispose()
})
