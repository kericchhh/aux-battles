import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate } from 'react-router'

import { createBattle, maxRounds, timerOptions, type GuessSeconds } from '@/api/battles'
import { FormError } from '@/components/form-error'
import { Button } from '@/components/ui/button'
import { Segmented } from '@/components/ui/segmented'

const rounds = Array.from({ length: maxRounds }, (_, i) => ({ value: i + 1, label: String(i + 1) }))
const timers = timerOptions.map((value) => ({
  value,
  label: value === null ? 'Off' : `${value}s`,
}))

const maxPoints = (count: number) =>
  Array.from({ length: count }, (_, i) => 100 + Math.min(10 * i, 20)).reduce((a, b) => a + b, 0)

export function HostForm() {
  const navigate = useNavigate()
  const [roundCount, setRoundCount] = useState(5)
  const [guessSeconds, setGuessSeconds] = useState<GuessSeconds>(null)
  const create = useMutation({
    mutationFn: createBattle,
    onSuccess: (battle) => navigate(`/battles/${battle.id}`),
  })

  return (
    <form
      className="grid gap-5"
      onSubmit={(event) => {
        event.preventDefault()
        create.mutate({ rounds: roundCount, guessSeconds })
      }}
    >
      <Segmented
        legend="Rounds"
        name="rounds"
        options={rounds}
        value={roundCount}
        onChange={setRoundCount}
        hint={
          <>
            Up to <span className="font-mono text-paper tabular">{maxPoints(roundCount)}</span>{' '}
            points
          </>
        }
      />
      <Segmented
        legend="Time per stem"
        name="guessSeconds"
        options={timers}
        value={guessSeconds}
        onChange={setGuessSeconds}
        hint="Running out of time counts as a wrong guess."
      />
      <FormError message={create.error?.message} />
      <Button type="submit" disabled={create.isPending}>
        {create.isPending ? 'Creating…' : 'Create battle'}
      </Button>
    </form>
  )
}
