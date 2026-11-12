import { act, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Countdown } from './countdown'
import { battle } from '@/test/fixtures'

afterEach(() => vi.useRealTimers())

it('announces the actual remaining time on a late join and does not repeat every second', async () => {
  vi.useFakeTimers()
  const now = Date.now()
  const client = new QueryClient()
  const invalidate = vi.spyOn(client, 'invalidateQueries')
  render(
    <QueryClientProvider client={client}>
      <Countdown
        battle={battle({ serverTime: new Date(now).toISOString() })}
        endsAt={new Date(now + 3000).toISOString()}
        seconds={15}
        receivedAt={now}
      />
    </QueryClientProvider>,
  )
  expect(screen.getByRole('status')).toHaveTextContent('3 seconds left.')
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1000)
  })
  expect(screen.getByRole('status')).toHaveTextContent('3 seconds left.')
  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000)
  })
  expect(screen.getByRole('status')).toHaveTextContent('Time is up')
  await act(async () => {
    await vi.advanceTimersByTimeAsync(7500)
  })
  expect(invalidate).toHaveBeenCalledTimes(3)
  expect(invalidate).toHaveBeenLastCalledWith(
    { queryKey: ['battles', 'b1'] },
    { cancelRefetch: false },
  )
})
