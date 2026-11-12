import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router'
import { z } from 'zod'
import { joinBattle } from '@/api/battles'
import { isApiError } from '@/api/client'
import { Field } from '@/components/field'
import { FormError } from '@/components/form-error'
import { Button } from '@/components/ui/button'

const schema = z.object({
  code: z
    .string()
    .trim()
    .toUpperCase()
    .regex(/^[A-Z0-9]{6}$/, 'Codes are 6 letters or numbers'),
})
function joinMessage(error: Error) {
  if (isApiError(error, 404)) return 'No battle uses that code.'
  if (isApiError(error, 409)) return 'That battle is full, or it is your own.'
  return error.message
}
export function JoinForm() {
  const navigate = useNavigate()
  const form = useForm({ resolver: zodResolver(schema) })
  const join = useMutation({
    mutationFn: joinBattle,
    onSuccess: (battle) => navigate(`/battles/${battle.id}`),
  })
  return (
    <form
      className="grid gap-5"
      noValidate
      onSubmit={form.handleSubmit(({ code }) => {
        if (!join.isPending) join.mutate(code)
      })}
    >
      <Field
        label="Invite code"
        {...form.register('code')}
        maxLength={6}
        autoComplete="off"
        autoCapitalize="characters"
        spellCheck={false}
        placeholder="ABC123"
        className="font-mono tracking-widest uppercase"
        hint="Ask the host for their 6-character code."
        error={form.formState.errors.code?.message}
      />
      <FormError message={join.error && joinMessage(join.error)} />
      <Button type="submit" disabled={join.isPending}>
        {join.isPending ? 'Joining…' : 'Join battle'}
      </Button>
    </form>
  )
}
