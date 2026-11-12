import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import type { Battle } from '@/api/battles'
import { axeViolations, json, mockApi, renderRoute } from '@/test/render'
import { BattlePage } from './battle'
import { battle, round, songs } from '@/test/fixtures'

function renderBattle(view: Battle, routes = {}) {
  const fetch = mockApi({ 'GET /battles/b1': () => json(view), ...routes })
  return { fetch, ...renderRoute(<BattlePage />, '/battles/b1', '/battles/:battleId') }
}

const body = (fetch: ReturnType<typeof mockApi>, method: string, path: string) => {
  const call = fetch.mock.calls.find(
    ([url, init]) => init?.method === method && String(url).endsWith(path),
  )
  return call && JSON.parse(String(call[1]?.body))
}

describe('BattlePage', () => {
  it('shows the invite code while waiting for an opponent', async () => {
    const { container } = renderBattle(
      battle({ status: 'PENDING', opponent: null, opponentJoined: false, round: null }),
    )

    expect(await screen.findByLabelText('Invite code')).toHaveTextContent('ABC123')
    expect(screen.getByRole('button', { name: 'Cancel lobby' })).toBeInTheDocument()
    expect(await axeViolations(container)).toEqual([])
  })

  it('locks an ordered lineup from the catalog', async () => {
    const { fetch, container } = renderBattle(
      battle({
        status: 'SELECTING',
        myLineupLocked: false,
        opponentLineupLocked: false,
        round: null,
      }),
      {
        'GET /songs': () => json({ songs, nextCursor: null }),
        'PUT /battles/b1/lineup': () => json({ battleId: 'b1', status: 'WAITING_ON_OPPONENT' }),
      },
    )

    await userEvent.click(await screen.findByRole('button', { name: 'Add Second Wind' }))
    await userEvent.click(screen.getByRole('button', { name: 'Add First Light' }))
    expect(screen.getByRole('button', { name: 'Add Third Rail' })).toHaveAttribute(
      'aria-disabled',
      'true',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Move First Light up' }))
    expect(await axeViolations(container)).toEqual([])
    await userEvent.click(screen.getByRole('button', { name: 'Lock lineup' }))

    await waitFor(() => expect(body(fetch, 'PUT', '/lineup')).toEqual({ songIds: ['s1', 's2'] }))
  })

  it('searches the catalog', async () => {
    const { fetch } = renderBattle(
      battle({ status: 'SELECTING', myLineupLocked: false, round: null }),
      { 'GET /songs': () => json({ songs: [], nextCursor: null }) },
    )

    await userEvent.type(await screen.findByLabelText('Search songs'), 'wind')

    expect(await screen.findByText('No songs match “wind”.')).toBeInTheDocument()
    expect(fetch.mock.calls.some(([url]) => String(url).includes('q=wind'))).toBe(true)
  })

  it('announces a wrong guess and sends the expected attempt', async () => {
    const { fetch, container } = renderBattle(battle(), {
      'POST /rounds/r1/guess': () =>
        json({
          battleId: 'b1',
          roundId: 'r1',
          attempt: 1,
          correct: false,
          result: 'INCORRECT',
          basePoints: 0,
          streakBonus: 0,
          pointsAwarded: 0,
          stage: 'BASS',
          playerFinished: false,
          roundFinished: false,
          battleStatus: 'ONGOING',
        }),
    })

    expect(await axeViolations(container)).toEqual([])
    await userEvent.type(await screen.findByLabelText('Song title'), 'Wrong song')
    await userEvent.click(screen.getByRole('button', { name: 'Guess' }))

    expect((await screen.findByText('Not it. Bass unlocked.')).parentElement).toHaveAttribute(
      'role',
      'status',
    )
    expect(body(fetch, 'POST', '/guess')).toEqual({ guess: 'Wrong song', expectedAttempt: 1 })
    expect(screen.getByLabelText('Song title')).toHaveValue('')
  })

  it('replaces stale feedback when a stem times out', async () => {
    let view = battle({ guessSeconds: 15 })
    mockApi({
      'GET /battles/b1': () => json(view),
      'POST /rounds/r1/guess': () => {
        view = battle({ guessSeconds: 15, round: { ...round, myAttempts: 1, myStage: 'BASS' } })
        return json({
          battleId: 'b1',
          roundId: 'r1',
          attempt: 1,
          correct: false,
          result: 'INCORRECT',
          basePoints: 0,
          streakBonus: 0,
          pointsAwarded: 0,
          stage: 'BASS',
          playerFinished: false,
          roundFinished: false,
          battleStatus: 'ONGOING',
        })
      },
    })
    const { queryClient } = renderRoute(<BattlePage />, '/battles/b1', '/battles/:battleId')

    await userEvent.type(await screen.findByLabelText('Song title'), 'Wrong song')
    await userEvent.click(screen.getByRole('button', { name: 'Guess' }))
    expect(await screen.findByText('Not it. Bass unlocked.')).toBeInTheDocument()

    view = battle({ guessSeconds: 15, round: { ...round, myAttempts: 2, myStage: 'MELODY' } })
    await queryClient.invalidateQueries()

    expect(await screen.findByText('Time ran out. Melody unlocked.')).toBeInTheDocument()
  })

  it('explains a guess that arrived after the time ran out', async () => {
    renderBattle(battle(), {
      'POST /rounds/r1/guess': () =>
        json({ message: 'time ran out on this stem; refresh the battle' }, 409),
    })

    await userEvent.type(await screen.findByLabelText('Song title'), 'Late')
    await userEvent.click(screen.getByRole('button', { name: 'Guess' }))

    expect(
      await screen.findByText('time ran out on this stem; refresh the battle'),
    ).toBeInTheDocument()
  })

  it('counts down using the server clock', async () => {
    const serverNow = Date.now() - 60_000
    renderBattle(
      battle({
        guessSeconds: 30,
        serverTime: new Date(serverNow).toISOString(),
        round: { ...round, stageEndsAt: new Date(serverNow + 20_000).toISOString() },
      }),
    )

    const timer = await screen.findByRole('timer')
    expect(timer.textContent).toMatch(/^0:(19|20)$/)
  })

  it('reveals the previous round', async () => {
    renderBattle(
      battle({
        currentRound: 2,
        myScore: 100,
        opponentScore: 50,
        round: { ...round, id: 'r2', number: 2 },
        completedRounds: [
          {
            number: 1,
            myPoints: 100,
            opponentPoints: 50,
            mySong: { title: 'First Light', artist: 'Ana' },
            opponentSong: { title: 'Second Wind', artist: 'Bea' },
          },
        ],
      }),
    )

    const reveal = await screen.findByRole('complementary', { name: 'Round 1 reveal' })
    expect(reveal).toHaveTextContent('Second Wind')
    expect(
      within(screen.getByRole('region', { name: 'Score' })).getByText('100'),
    ).toBeInTheDocument()
  })

  it('shows the result of a finished battle', async () => {
    const { container } = renderBattle(
      battle({
        status: 'FINISHED',
        outcome: 'WIN',
        endReason: 'FORFEIT',
        myScore: 100,
        round: null,
      }),
    )

    expect(await screen.findByRole('heading', { name: 'You won' })).toBeInTheDocument()
    expect(screen.getByText('bea forfeited.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Forfeit' })).not.toBeInTheDocument()
    expect(await axeViolations(container)).toEqual([])
  })

  it('forfeits after confirmation', async () => {
    const ended = battle({ status: 'FINISHED', outcome: 'LOSS', endReason: 'FORFEIT', round: null })
    const { fetch } = renderBattle(battle(), { 'POST /battles/b1/forfeit': () => json(ended) })

    await userEvent.click(await screen.findByRole('button', { name: 'Forfeit' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Forfeit this battle?' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Forfeit' }))

    expect(await screen.findByRole('heading', { name: 'You lost' })).toBeInTheDocument()
    expect(fetch.mock.calls.some(([url]) => String(url).endsWith('/forfeit'))).toBe(true)
  })

  it('keeps the game on screen when a refresh fails', async () => {
    let down = false
    mockApi({ 'GET /battles/b1': () => (down ? json({ message: 'boom' }, 500) : json(battle())) })
    const { queryClient } = renderRoute(<BattlePage />, '/battles/b1', '/battles/:battleId')
    await screen.findByLabelText('Song title')

    down = true
    await queryClient.invalidateQueries()

    expect(await screen.findByText('Could not refresh the battle. Retrying…')).toBeInTheDocument()
    expect(screen.getByLabelText('Song title')).toBeInTheDocument()
  })

  it('does not blame the timer for a guess scored elsewhere', async () => {
    let view = battle({ guessSeconds: 15 })
    mockApi({ 'GET /battles/b1': () => json(view) })
    const { queryClient } = renderRoute(<BattlePage />, '/battles/b1', '/battles/:battleId')
    await screen.findByLabelText('Song title')

    view = battle({
      guessSeconds: 15,
      myScore: 100,
      round: { ...round, myAttempts: 1, myPoints: 100, myFinished: true },
    })
    await queryClient.invalidateQueries()

    expect(await screen.findByText(/You scored/)).toBeInTheDocument()
    expect(screen.queryByText(/Time ran out/)).not.toBeInTheDocument()
  })

  it('says when the stem audio cannot be loaded', async () => {
    const created: HTMLAudioElement[] = []
    const RealAudio = globalThis.Audio
    vi.stubGlobal(
      'Audio',
      class extends RealAudio {
        constructor(src?: string) {
          super(src)
          created.push(this)
        }
      },
    )
    renderBattle(battle())
    await screen.findByRole('group', { name: 'Listen to the drums stem' })

    const audio = created.at(-1)!
    Object.defineProperty(audio, 'error', { value: { code: 4 } })
    act(() => void audio.dispatchEvent(new Event('error')))

    expect(await screen.findByText('This audio could not be loaded.')).toBeInTheDocument()
  })

  it('explains a missing battle', async () => {
    mockApi({})
    renderRoute(<BattlePage />, '/battles/b1', '/battles/:battleId')

    expect(await screen.findByRole('heading', { name: 'Battle not found' })).toBeInTheDocument()
  })
  it('shows the actual rejection reason for unavailable audio', async () => {
    renderBattle(battle(), {
      'POST /rounds/r1/guess': () => json({ message: 'round audio is unavailable' }, 409),
    })
    await userEvent.type(await screen.findByLabelText('Song title'), 'Title')
    await userEvent.click(screen.getByRole('button', { name: 'Guess' }))
    expect(await screen.findByText('round audio is unavailable')).toBeInTheDocument()
    expect(screen.queryByText(/out of date/)).not.toBeInTheDocument()
  })

  it('moves focus to the stage when a scored guess removes its form', async () => {
    let view = battle()
    mockApi({
      'GET /battles/b1': () => json(view),
      'POST /rounds/r1/guess': () => {
        view = battle({ round: { ...round, myFinished: true, myPoints: 100, myAttempts: 1 } })
        return json({
          attempt: 1,
          correct: true,
          stage: 'DRUM',
          pointsAwarded: 100,
          streakBonus: 0,
          playerFinished: true,
        })
      },
    })
    renderRoute(<BattlePage />, '/battles/b1', '/battles/:battleId')
    await userEvent.type(await screen.findByLabelText('Song title'), 'Correct title')
    await userEvent.click(screen.getByRole('button', { name: 'Guess' }))
    const region = await screen.findByRole('region', {
      name: 'Round 1 complete for you. Waiting for your opponent.',
    })
    expect(region).toHaveFocus()
    expect(screen.queryByLabelText('Song title')).not.toBeInTheDocument()
  })

  it('moves focus and announces the locked lineup after its controls disappear', async () => {
    let view = battle({ status: 'SELECTING', rounds: 1, myLineupLocked: false, round: null })
    mockApi({
      'GET /battles/b1': () => json(view),
      'GET /songs': () => json({ songs, nextCursor: null }),
      'PUT /battles/b1/lineup': () => {
        view = { ...view, myLineupLocked: true }
        return json({ status: 'WAITING_ON_OPPONENT' })
      },
    })
    renderRoute(<BattlePage />, '/battles/b1', '/battles/:battleId')
    await userEvent.click(await screen.findByRole('button', { name: 'Add First Light' }))
    await userEvent.click(screen.getByRole('button', { name: 'Lock lineup' }))
    const region = await screen.findByRole('region', {
      name: 'Lineup locked. Waiting for the first round.',
    })
    expect(region).toHaveFocus()
    expect(within(region).getByText('Lineup locked. Waiting for the first round.')).toHaveAttribute(
      'role',
      'status',
    )
  })
})
