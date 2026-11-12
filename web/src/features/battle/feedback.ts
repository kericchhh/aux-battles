import type { GuessResult, RoundView } from '@/api/battles'

import { stages } from '@/components/game/stages'

export type Feedback = { attempts: number; tone: 'good' | 'bad'; text: string }

export function guessFeedback(result: GuessResult): Feedback {
  if (result.correct) {
    const bonus = result.streakBonus ? `, including a ${result.streakBonus}-point streak bonus` : ''
    return {
      attempts: result.attempt,
      tone: 'good',
      text: `Correct! +${result.pointsAwarded} points${bonus}.`,
    }
  }
  const next = stages.find((s) => s.id === result.stage)
  const text = result.playerFinished
    ? 'Not it. No guesses left, so 0 points this round.'
    : next
      ? `Not it. ${next.label} unlocked.`
      : 'Not it.'
  return { attempts: result.attempt, tone: 'bad', text }
}

export function timeoutFeedback(round: RoundView): Feedback {
  const stage = stages.find((s) => s.id === round.myStage)
  const text = round.myFinished
    ? 'Time ran out. No guesses left this round.'
    : stage
      ? `Time ran out. ${stage.label} unlocked.`
      : 'Time ran out.'
  return { attempts: round.myAttempts, tone: 'bad', text }
}

export function rejectedFeedback(attempts: number, message: string): Feedback {
  const text = message || 'Your guess could not be accepted. Refresh the battle and try again.'
  return { attempts, tone: 'bad', text }
}
