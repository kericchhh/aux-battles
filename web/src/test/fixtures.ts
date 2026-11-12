import type { Battle, RoundView } from '@/api/battles'

export const round: RoundView = {
  id: 'r1',
  number: 1,
  status: 'GUESSING',
  myStage: 'DRUM',
  myPoints: 0,
  myFinished: false,
  opponentFinished: false,
  myAttempts: 0,
  stageEndsAt: null,
}

export function battle(overrides: Partial<Battle> = {}): Battle {
  return {
    id: 'b1',
    status: 'ONGOING',
    inviteCode: 'ABC123',
    currentRound: 1,
    rounds: 2,
    guessSeconds: null,
    me: { username: 'ana' },
    opponent: { username: 'bea' },
    opponentJoined: true,
    myLineupLocked: true,
    opponentLineupLocked: true,
    outcome: null,
    endReason: null,
    myScore: 0,
    opponentScore: 0,
    round,
    completedRounds: [],
    serverTime: new Date().toISOString(),
    ...overrides,
  }
}

export const songs = [
  { id: 's1', title: 'First Light', artist: 'Ana' },
  { id: 's2', title: 'Second Wind', artist: 'Bea' },
  { id: 's3', title: 'Third Rail', artist: 'Cy' },
]
