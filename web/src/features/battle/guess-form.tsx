import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { battleKey, guessKey, submitGuess, type RoundView } from '@/api/battles'
import { isApiError } from '@/api/client'
import { Field } from '@/components/field'
import { FormError } from '@/components/form-error'
import { Button } from '@/components/ui/button'
import { guessFeedback, rejectedFeedback, type Feedback } from './feedback'

const schema = z.object({
  guess: z.string().trim().min(1, 'Type a song title to guess').max(255, 'At most 255 characters'),
})
type GuessFormProps = {
  battleId: string
  round: RoundView
  onFeedback: (feedback: Feedback) => void
}
export function GuessForm({ battleId, round, onFeedback }: GuessFormProps) {
  const queryClient = useQueryClient()
  const form = useForm({ resolver: zodResolver(schema), defaultValues: { guess: '' } })
  const [lastAttempt, setLastAttempt] = useState(0)
  const attempt = Math.max(round.myAttempts, lastAttempt) + 1
  const send = useMutation({
    mutationKey: guessKey(round.id),
    mutationFn: (title: string) => submitGuess(round.id, title, attempt),
    onSuccess: (result) => {
      setLastAttempt(result.attempt)
      onFeedback(guessFeedback(result))
      form.reset()
      form.setFocus('guess')
    },
    onError: (error) => {
      if (isApiError(error, 409)) onFeedback(rejectedFeedback(attempt - 1, error.message))
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: battleKey(battleId) }),
  })
  return (
    <form
      noValidate
      onSubmit={form.handleSubmit(({ guess }) => {
        if (!send.isPending) send.mutate(guess)
      })}
      className="grid gap-4"
    >
      <Field
        label="Song title"
        {...form.register('guess')}
        maxLength={255}
        autoComplete="off"
        spellCheck={false}
        error={form.formState.errors.guess?.message}
      />
      <FormError message={send.error && !isApiError(send.error, 409) ? send.error.message : null} />
      <Button type="submit" disabled={send.isPending}>
        {send.isPending ? 'Checking…' : 'Guess'}
      </Button>
    </form>
  )
}
