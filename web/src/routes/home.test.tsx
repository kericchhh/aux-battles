import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import type { BattleSummary, Stats } from '@/api/battles'
import { axeViolations, json, mockApi, renderRoute } from '@/test/render'
import { Home } from './home'

const stats: Stats = { played: 4, wins: 2, losses: 1, draws: 1, bestScore: 330, averageScore: 210 }

const summary: BattleSummary = {
  id: 'b2',
  status: 'ONGOING',
  rounds: 3,
  currentRound: 2,
  opponent: { username: 'bea' },
  inviteCode: null,
  myScore: 100,
  opponentScore: 75,
  outcome: null,
  endReason: null,
  lastActivityAt: '2026-11-11T10:00:00Z',
}

function homeApi(battles: BattleSummary[] = [], extra = {}) {
  return mockApi({
    'GET /battles': () => json(battles),
    'GET /users/me/stats': () => json(stats),
    ...extra,
  })
}

describe('Home', () => {
  it('has no detectable accessibility violations', async () => {
    homeApi([summary])
    const { container } = renderRoute(<Home />)
    await screen.findByRole('link', { name: /vs bea/ })
    expect(await axeViolations(container)).toEqual([])
  })

  it('creates a battle with the chosen round count and timer', async () => {
    const fetch = homeApi([], {
      'POST /battles': () => json({ id: 'b1', inviteCode: 'ABC123' }, 201),
    })
    const { router } = renderRoute(<Home />)

    await userEvent.click(screen.getByRole('radio', { name: '3' }))
    await userEvent.click(screen.getByRole('radio', { name: '15s' }))
    await userEvent.click(screen.getByRole('button', { name: 'Create battle' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/battles/b1'))
    const create = fetch.mock.calls.find(([, init]) => init?.method === 'POST')!
    expect(JSON.parse(String(create[1]?.body))).toEqual({ rounds: 3, guessSeconds: 15 })
  })

  it('creates untimed battles by default', async () => {
    const fetch = homeApi([], {
      'POST /battles': () => json({ id: 'b1', inviteCode: 'ABC123' }, 201),
    })
    renderRoute(<Home />)

    expect(screen.getByRole('radio', { name: 'Off' })).toBeChecked()
    await userEvent.click(screen.getByRole('button', { name: 'Create battle' }))

    await waitFor(() =>
      expect(fetch.mock.calls.some(([, init]) => init?.method === 'POST')).toBe(true),
    )
    const create = fetch.mock.calls.find(([, init]) => init?.method === 'POST')!
    expect(JSON.parse(String(create[1]?.body))).toEqual({ rounds: 5, guessSeconds: null })
  })

  it('shows the most points the chosen round count allows', async () => {
    homeApi()
    renderRoute(<Home />)

    await userEvent.click(screen.getByRole('radio', { name: '3' }))

    expect(screen.getByText(/Up to/)).toHaveTextContent('Up to 330 points')
  })

  it('shows player stats', async () => {
    homeApi()
    renderRoute(<Home />)

    const card = screen.getByRole('region', { name: 'Your stats' })
    await waitFor(() =>
      expect(within(card).getByText('Best score').nextSibling).toHaveTextContent('330'),
    )
    expect(within(card).getByText('Wins').nextSibling).toHaveTextContent('2')
  })

  it('lists battles with their progress', async () => {
    homeApi([summary])
    renderRoute(<Home />)

    const link = await screen.findByRole('link', { name: /vs bea/ })
    expect(link).toHaveAttribute('href', '/battles/b2')
    expect(link).toHaveTextContent('Round 2 of 3')
    expect(within(link).getByText('You 100, them 75')).toBeInTheDocument()
  })

  it('keeps the battle list when a refresh fails', async () => {
    let down = false
    mockApi({
      'GET /battles': () => (down ? json({ message: 'boom' }, 500) : json([summary])),
      'GET /users/me/stats': () => (down ? json({ message: 'boom' }, 500) : json(stats)),
    })
    const { queryClient } = renderRoute(<Home />)
    await screen.findByRole('link', { name: /vs bea/ })

    down = true
    await queryClient.invalidateQueries()

    expect(screen.getByRole('link', { name: /vs bea/ })).toBeInTheDocument()
    expect(screen.queryByText('Could not load your battles.')).not.toBeInTheDocument()
    expect(screen.queryByText('Could not load your stats.')).not.toBeInTheDocument()
  })

  it('reports battles that could not be loaded', async () => {
    mockApi({ 'GET /users/me/stats': () => json(stats) })
    renderRoute(<Home />)

    expect(await screen.findByText('Could not load your battles.')).toBeInTheDocument()
  })

  it('explains the game when there are no battles yet', async () => {
    homeApi()
    renderRoute(<Home />)

    expect(await screen.findByRole('heading', { name: 'How it works' })).toBeInTheDocument()
  })

  it('rejects malformed invite codes without calling the API', async () => {
    const fetch = homeApi()
    renderRoute(<Home />)

    await userEvent.click(screen.getByRole('tab', { name: 'Join' }))
    await userEvent.type(screen.getByLabelText('Invite code'), 'ab1')
    await userEvent.click(screen.getByRole('button', { name: 'Join battle' }))

    expect(screen.getByLabelText('Invite code')).toHaveAccessibleDescription(
      'Ask the host for their 6-character code. Codes are 6 letters or numbers',
    )
    expect(fetch.mock.calls.some(([url]) => String(url).includes('/join'))).toBe(false)
  })

  it('explains an unknown invite code', async () => {
    homeApi([], { 'POST /battles/join': () => json({ message: 'battle not found' }, 404) })
    renderRoute(<Home />)

    await userEvent.click(screen.getByRole('tab', { name: 'Join' }))
    await userEvent.type(screen.getByLabelText('Invite code'), 'zzz999')
    await userEvent.click(screen.getByRole('button', { name: 'Join battle' }))

    expect(await screen.findByText('No battle uses that code.')).toBeInTheDocument()
  })
})
