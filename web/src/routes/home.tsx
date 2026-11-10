import { useMutation } from '@tanstack/react-query'
import { useState, type CSSProperties, type FormEvent } from 'react'
import { useNavigate } from 'react-router'

import { createBattle, joinBattle, maxRounds } from '@/api/battles'
import { isApiError } from '@/api/client'
import { Field } from '@/components/field'
import { FormError } from '@/components/form-error'
import { StemMeter } from '@/components/stem-meter'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

const roundOptions = Array.from({ length: maxRounds }, (_, i) => i + 1)
const inviteCodePattern = /^[A-Z0-9]{6}$/
const modes = ['host', 'join'] as const
type Mode = (typeof modes)[number]

const maxPoints = (rounds: number) =>
  roundOptions.slice(0, rounds).reduce((sum, n) => sum + 100 + Math.min(10 * (n - 1), 20), 0)

export function Home() {
  return (
    <div className="grid gap-8">
      <div className="page-header">
        <h1 className="text-3xl">Play</h1>
        <p>Name your opponent’s song from as few stems as you can.</p>
      </div>
      <div className="grid gap-5 lg:grid-cols-12">
        <StartCard />
        <HowItWorks />
      </div>
    </div>
  )
}

function StartCard() {
  const [mode, setMode] = useState<Mode>('host')

  return (
    <section aria-labelledby="start-title" className="card lg:col-span-5">
      <h2 id="start-title" className="text-lg">
        Start
      </h2>
      <Tabs value={mode} onValueChange={(value) => setMode(value as Mode)}>
        <TabsList aria-label="Start a battle" count={modes.length} selected={modes.indexOf(mode)}>
          <TabsTrigger value="host">Host</TabsTrigger>
          <TabsTrigger value="join">Join</TabsTrigger>
        </TabsList>
        <TabsContent value="host" className="tab-panel">
          <HostForm />
        </TabsContent>
        <TabsContent value="join" className="tab-panel">
          <JoinForm />
        </TabsContent>
      </Tabs>
    </section>
  )
}

function HostForm() {
  const navigate = useNavigate()
  const [rounds, setRounds] = useState(5)
  const create = useMutation({
    mutationFn: createBattle,
    onSuccess: (battle) => navigate(`/battles/${battle.id}`),
  })

  return (
    <form
      className="grid gap-5"
      onSubmit={(event) => {
        event.preventDefault()
        create.mutate(rounds)
      }}
    >
      <fieldset className="grid gap-2">
        <legend className="label mb-2">Rounds</legend>
        <div className="segmented" style={{ '--selected': rounds - 1 } as CSSProperties}>
          <span className="segmented-thumb" aria-hidden />
          {roundOptions.map((n) => (
            <label key={n}>
              <input
                type="radio"
                name="rounds"
                value={n}
                checked={rounds === n}
                onChange={() => setRounds(n)}
                className="sr-only"
              />
              <span>{n}</span>
            </label>
          ))}
        </div>
        <p className="text-dim">
          Up to <span className="font-mono text-paper tabular">{maxPoints(rounds)}</span> points
        </p>
      </fieldset>
      <FormError message={create.error?.message} />
      <Button type="submit" disabled={create.isPending}>
        {create.isPending ? 'Creating…' : 'Create battle'}
      </Button>
    </form>
  )
}

function JoinForm() {
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [invalid, setInvalid] = useState(false)
  const join = useMutation({
    mutationFn: joinBattle,
    onSuccess: (battle) => navigate(`/battles/${battle.id}`),
  })

  function submit(event: FormEvent) {
    event.preventDefault()
    const normalized = code.trim().toUpperCase()
    setInvalid(!inviteCodePattern.test(normalized))
    if (inviteCodePattern.test(normalized)) join.mutate(normalized)
  }

  return (
    <form className="grid gap-5" noValidate onSubmit={submit}>
      <Field
        label="Invite code"
        value={code}
        onChange={(event) => setCode(event.target.value.toUpperCase())}
        maxLength={6}
        autoComplete="off"
        autoCapitalize="characters"
        spellCheck={false}
        placeholder="ABC123"
        className="font-mono tracking-widest uppercase"
        hint="Ask the host for their 6-character code."
        error={invalid ? 'Codes are 6 letters or numbers' : undefined}
      />
      <FormError message={join.error && joinMessage(join.error)} />
      <Button type="submit" disabled={join.isPending}>
        {join.isPending ? 'Joining…' : 'Join battle'}
      </Button>
    </form>
  )
}

function joinMessage(error: Error) {
  if (isApiError(error, 404)) return 'No battle uses that code.'
  if (isApiError(error, 409)) return 'That battle is full, or it is your own.'
  return error.message
}

function HowItWorks() {
  return (
    <section aria-labelledby="rules-title" className="card lg:col-span-7">
      <h2 id="rules-title" className="text-lg">
        How it works
      </h2>
      <ol className="steps">
        <li>Host a battle of 1–7 rounds and share the invite code.</li>
        <li>Both players secretly pick one song per round.</li>
        <li>
          Each round you hear your opponent’s song one stem at a time. A wrong guess unlocks the
          next stem, up to four guesses.
        </li>
        <li>
          Earlier guesses score more. Consecutive correct rounds add 10 bonus points each, up to 20.
        </li>
      </ol>
      <div className="grid gap-2">
        <p className="text-dim">Points for a correct guess at each stem</p>
        <StemMeter className="stem-meter-inset" />
      </div>
    </section>
  )
}
