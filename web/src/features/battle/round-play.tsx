import { useIsMutating } from '@tanstack/react-query'
import { useState } from 'react'

import { guessKey, opponentName, roundAudioUrl, type Battle, type RoundView } from '@/api/battles'
import { AudioPlayer } from '@/components/audio-player'
import { Pending } from '@/components/pending'
import { Card } from '@/components/ui/card'
import { useAudio } from '@/hooks/use-audio'

import { Countdown } from './countdown'
import { timeoutFeedback, type Feedback } from './feedback'
import { GuessForm } from './guess-form'
import { RoundReveal } from './round-reveal'
import { stages } from '@/components/game/stages'
import { StemMeter } from '@/components/game/stem-meter'

type RoundPlayProps = { battle: Battle; receivedAt: number }

export function RoundPlay({ battle, receivedAt }: RoundPlayProps) {
  if (!battle.round) return <Pending>Loading the next round…</Pending>
  return (
    <div className="grid items-start gap-5 lg:grid-cols-12">
      <RoundCard
        key={battle.round.id}
        battle={battle}
        round={battle.round}
        receivedAt={receivedAt}
      />
      <RoundReveal battle={battle} className="lg:col-span-5" />
    </div>
  )
}

function RoundCard({ battle, round, receivedAt }: RoundPlayProps & { round: RoundView }) {
  const [seen] = useState(round.myAttempts)
  const [given, setGiven] = useState<Feedback | null>(null)
  const audio = useAudio(
    roundAudioUrl(round.id, round.myStage),
    round.myAttempts > 0 && !round.myFinished,
  )
  const stage = stages.find((s) => s.id === round.myStage)!
  const guessing = useIsMutating({ mutationKey: guessKey(round.id) }) > 0
  const timedOut =
    battle.guessSeconds &&
    !guessing &&
    round.myPoints === 0 &&
    round.myAttempts > Math.max(seen, given?.attempts ?? 0)
  const feedback = timedOut ? timeoutFeedback(round) : given

  return (
    <Card
      title={`Round ${round.number}`}
      action={
        <p className="text-dim">
          {round.myFinished ? 'Done' : `Guess ${round.myAttempts + 1} of 4`}
        </p>
      }
      className="lg:col-span-7"
    >
      <StemMeter current={round.myStage} done={round.myFinished} playing={audio.playing} />
      <AudioPlayer
        label={`${round.myFinished ? 'Replay' : 'Listen to'} the ${stage.label.toLowerCase()} stem`}
        audio={audio}
      />
      {battle.guessSeconds && round.stageEndsAt && !round.myFinished && (
        <Countdown
          battle={battle}
          endsAt={round.stageEndsAt}
          seconds={battle.guessSeconds}
          receivedAt={receivedAt}
        />
      )}
      <div role="status" className="live">
        {feedback && (
          <p className="feedback" data-tone={feedback.tone}>
            {feedback.text}
          </p>
        )}
      </div>
      {round.myFinished ? (
        <Pending>
          You scored <span className="font-mono text-paper tabular">{round.myPoints}</span> this
          round.{' '}
          {round.opponentFinished
            ? 'Starting the next round…'
            : `Waiting for ${opponentName(battle)} to finish…`}
        </Pending>
      ) : (
        <GuessForm battleId={battle.id} round={round} onFeedback={setGiven} />
      )}
    </Card>
  )
}
