import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'

import { useAudio } from './use-audio'
import { useAudioPreview } from './use-audio-preview'
import { useCountdown } from './use-countdown'
import { useDebounced } from './use-debounced'
import { useLiveBattle } from './use-live-battle'
import { battle } from '@/test/fixtures'
import { json, mockFetch } from '@/test/render'

class TestAudio extends EventTarget {
  static instances: TestAudio[] = []
  paused = true
  ended = false
  error: object | null = null
  currentTime = 0
  duration = 12
  preload = ''
  play = vi.fn(() => {
    this.paused = false
    this.dispatchEvent(new Event('play'))
    return Promise.resolve()
  })
  pause = vi.fn(() => {
    this.paused = true
    this.dispatchEvent(new Event('pause'))
  })
  removeAttribute = vi.fn()
  load = vi.fn()
  src: string
  constructor(src: string) {
    super()
    this.src = src
    TestAudio.instances.push(this)
  }
}

class TestSource extends EventTarget {
  static OPEN = 1
  static instances: TestSource[] = []
  readyState = 1
  onerror: (() => void) | null = null
  close = vi.fn()
  url: string
  constructor(url: string) {
    super()
    this.url = url
    TestSource.instances.push(this)
  }
}

afterEach(() => vi.useRealTimers())

it('does not reload unchanged audio when autoplay changes and releases downloads on cleanup', () => {
  TestAudio.instances = []
  vi.stubGlobal('Audio', TestAudio)
  const { result, rerender, unmount } = renderHook(({ auto }) => useAudio('/stem', auto), {
    initialProps: { auto: true },
  })
  const element = TestAudio.instances[0]!
  act(() => {
    element.currentTime = 4
    element.dispatchEvent(new Event('timeupdate'))
  })
  rerender({ auto: false })
  expect(TestAudio.instances).toHaveLength(1)
  expect(result.current.time).toBe(4)
  act(() => {
    element.error = {}
    element.dispatchEvent(new Event('error'))
  })
  expect(result.current.failed).toBe(true)
  unmount()
  expect(element.pause).toHaveBeenCalled()
  expect(element.removeAttribute).toHaveBeenCalledWith('src')
  expect(element.load).toHaveBeenCalled()
})

it('stops the previous preview and reports audio errors', () => {
  TestAudio.instances = []
  vi.stubGlobal('Audio', TestAudio)
  const { result } = renderHook(() => useAudioPreview((id) => `/songs/${id}`))
  act(() => result.current.toggle('first'))
  const first = TestAudio.instances[0]!
  act(() => result.current.toggle('second'))
  expect(first.pause).toHaveBeenCalled()
  expect(first.removeAttribute).toHaveBeenCalledWith('src')
  const second = TestAudio.instances[1]!
  act(() => {
    second.error = {}
    second.dispatchEvent(new Event('error'))
  })
  expect(result.current.playing).toBeNull()
  expect(result.current.failed).toBe('second')
  act(() => result.current.toggle('second'))
  expect(TestAudio.instances).toHaveLength(3)
  expect(TestAudio.instances[2]!.play).toHaveBeenCalledOnce()
  expect(result.current.failed).toBeNull()
})

it('uses the server clock and cancels countdown/debounce timers on unmount', async () => {
  vi.useFakeTimers()
  const now = Date.now()
  const countdown = renderHook(() =>
    useCountdown(
      new Date(now - 60_000 + 3000).toISOString(),
      new Date(now - 60_000).toISOString(),
      now,
    ),
  )
  expect(countdown.result.current).toBe(3000)
  const debounced = renderHook(({ value }) => useDebounced(value, 250), {
    initialProps: { value: 'first' },
  })
  debounced.rerender({ value: 'second' })
  expect(debounced.result.current).toBe('first')
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1000)
  })
  expect(debounced.result.current).toBe('second')
  expect(countdown.result.current).toBe(2000)
  debounced.unmount()
  countdown.unmount()
  expect(vi.getTimerCount()).toBe(0)
})

it('refreshes on live events and closes the stream when a battle ends', async () => {
  TestSource.instances = []
  vi.stubGlobal('EventSource', TestSource)
  let view = battle()
  const fetch = mockFetch(() => json(view))
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  const { result, unmount } = renderHook(() => useLiveBattle('b1'), { wrapper })
  await waitFor(() => expect(result.current.battle.isSuccess).toBe(true))
  const source = TestSource.instances[0]!
  expect(source.url).toBe('/api/battles/b1/events')
  view = battle({ status: 'FINISHED', round: null })
  act(() => source.dispatchEvent(new Event('changed')))
  await waitFor(() => expect(result.current.battle.data?.status).toBe('FINISHED'))
  expect(fetch).toHaveBeenCalledTimes(2)
  expect(source.close).toHaveBeenCalled()
  unmount()
  client.clear()
})
