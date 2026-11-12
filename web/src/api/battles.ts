import { queryOptions } from '@tanstack/react-query'

import { api } from './client'

export type BattleStatus = 'PENDING' | 'SELECTING' | 'ONGOING' | 'FINISHED' | 'CANCELLED'
export type Stage = 'DRUM' | 'BASS' | 'MELODY' | 'FULL'
export type Outcome = 'WIN' | 'LOSS' | 'DRAW'
export type EndReason = 'COMPLETED' | 'FORFEIT' | 'TIMEOUT'
export type Player = { username: string }
export type SongInfo = { title: string; artist: string }

export type RoundView = {
  id: string
  number: number
  status: 'WAITING' | 'GUESSING' | 'FINISHED'
  myStage: Stage
  myPoints: number
  myFinished: boolean
  opponentFinished: boolean
  myAttempts: number
  stageEndsAt: string | null
}

export type CompletedRound = {
  number: number
  myPoints: number
  opponentPoints: number
  mySong: SongInfo
  opponentSong: SongInfo
}

export type Battle = {
  id: string
  status: BattleStatus
  inviteCode: string
  currentRound: number
  rounds: number
  guessSeconds: number | null
  me: Player
  opponent: Player | null
  opponentJoined: boolean
  myLineupLocked: boolean
  opponentLineupLocked: boolean
  outcome: Outcome | null
  endReason: EndReason | null
  myScore: number
  opponentScore: number
  round: RoundView | null
  completedRounds: CompletedRound[]
  serverTime: string
}

export type BattleSummary = {
  id: string
  status: BattleStatus
  rounds: number
  currentRound: number
  opponent: Player | null
  inviteCode: string | null
  myScore: number
  opponentScore: number
  outcome: Outcome | null
  endReason: EndReason | null
  lastActivityAt: string
}

export type Stats = {
  played: number
  wins: number
  losses: number
  draws: number
  bestScore: number
  averageScore: number
}

export type GuessResult = {
  battleId: string
  roundId: string
  attempt: number
  correct: boolean
  result: 'CORRECT' | 'INCORRECT' | 'TIMED_OUT'
  basePoints: number
  streakBonus: number
  pointsAwarded: number
  stage: Stage
  playerFinished: boolean
  roundFinished: boolean
  battleStatus: BattleStatus
}

export const maxRounds = 7
export const timerOptions = [null, 30, 15] as const
export type GuessSeconds = (typeof timerOptions)[number]

export const isActive = (status: BattleStatus) =>
  status === 'PENDING' || status === 'SELECTING' || status === 'ONGOING'

export const battleKey = (id: string) => ['battles', id] as const

export const battleQuery = (id: string) =>
  queryOptions({
    queryKey: battleKey(id),
    queryFn: () => api<Battle>(`/battles/${encodeURIComponent(id)}`),
  })

export const battlesQuery = queryOptions({
  queryKey: ['battles'],
  queryFn: () => api<BattleSummary[]>('/battles'),
  refetchInterval: 15_000,
})

export const statsQuery = queryOptions({
  queryKey: ['stats'],
  queryFn: () => api<Stats>('/users/me/stats'),
})

export const createBattle = (body: { rounds: number; guessSeconds: GuessSeconds }) =>
  api<{ id: string; inviteCode: string }>('/battles', {
    method: 'POST',
    body: JSON.stringify(body),
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

export const guessKey = (roundId: string) => ['rounds', roundId, 'guess'] as const

export const submitGuess = (roundId: string, guess: string, expectedAttempt: number) =>
  api<GuessResult>(`/rounds/${encodeURIComponent(roundId)}/guess`, {
    method: 'POST',
    body: JSON.stringify({ guess, expectedAttempt }),
  })

export const battleEventsUrl = (battleId: string) =>
  `/api/battles/${encodeURIComponent(battleId)}/events`

export const roundAudioUrl = (roundId: string, stage: Stage) =>
  `/api/rounds/${encodeURIComponent(roundId)}/audio?v=${stage}`

export function opponentName(battle: { opponent: Player | null }, fallback = 'your opponent') {
  return battle.opponent?.username ?? fallback
}
