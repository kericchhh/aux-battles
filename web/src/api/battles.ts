import { queryOptions } from '@tanstack/react-query'

import { api } from './client'

export type BattleStatus = 'PENDING' | 'SELECTING' | 'ONGOING' | 'FINISHED' | 'CANCELLED'
export type Stage = 'DRUM' | 'BASS' | 'MELODY' | 'FULL'

export type Battle = {
  id: string
  status: BattleStatus
  inviteCode: string
  currentRound: number
  rounds: number
  opponentJoined: boolean
  myLineupLocked: boolean
  opponentLineupLocked: boolean
  outcome: 'WIN' | 'LOSS' | 'DRAW' | null
  endReason: 'COMPLETED' | 'FORFEIT' | 'TIMEOUT' | null
  myScore: number
  opponentScore: number
  round: {
    id: string
    number: number
    status: 'WAITING' | 'GUESSING' | 'FINISHED'
    myStage: Stage
    myPoints: number
    myFinished: boolean
    opponentFinished: boolean
    myAttempts: number
  } | null
  previousRound: { number: number; myPoints: number; opponentPoints: number } | null
}

export type GuessResult = {
  battleId: string
  roundId: string
  attempt: number
  correct: boolean
  result: 'CORRECT' | 'INCORRECT'
  basePoints: number
  streakBonus: number
  pointsAwarded: number
  stage: Stage
  playerFinished: boolean
  roundFinished: boolean
  battleStatus: BattleStatus
}

export const maxRounds = 7

export const isActive = (status: BattleStatus) =>
  status === 'PENDING' || status === 'SELECTING' || status === 'ONGOING'

export const battleQuery = (id: string) =>
  queryOptions({
    queryKey: ['battles', id],
    queryFn: () => api<Battle>(`/battles/${encodeURIComponent(id)}`),
    refetchInterval: (query) => {
      const battle = query.state.data
      return battle && !isActive(battle.status) ? false : 2000
    },
  })

export const createBattle = (rounds: number) =>
  api<{ id: string; inviteCode: string }>('/battles', {
    method: 'POST',
    body: JSON.stringify({ rounds }),
  })

export const joinBattle = (inviteCode: string) =>
  api<{ id: string }>('/battles/join', {
    method: 'POST',
    body: JSON.stringify({ inviteCode }),
  })

export const lockLineup = (battleId: string, songIds: string[]) =>
  api<{ battleId: string; status: string }>(`/battles/${encodeURIComponent(battleId)}/lineup`, {
    method: 'PUT',
    body: JSON.stringify({ songIds }),
  })

export const forfeitBattle = (battleId: string) =>
  api<Battle>(`/battles/${encodeURIComponent(battleId)}/forfeit`, { method: 'POST' })

export const submitGuess = (roundId: string, guess: string, expectedAttempt: number) =>
  api<GuessResult>(`/rounds/${encodeURIComponent(roundId)}/guess`, {
    method: 'POST',
    body: JSON.stringify({ guess, expectedAttempt }),
  })

export const roundAudioUrl = (roundId: string, stage: Stage) =>
  `/api/rounds/${encodeURIComponent(roundId)}/audio?v=${stage}`
