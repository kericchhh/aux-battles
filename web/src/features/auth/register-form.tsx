import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router'
import { z } from 'zod'

import { acceptSession, register, signIn } from '@/api/auth'
import { isApiError } from '@/api/client'
import { Field } from '@/components/field'
import { FormError } from '@/components/form-error'
import { Button } from '@/components/ui/button'

const schema = z.object({
  username: z
    .string()
    .trim()
    .toLowerCase()
    .regex(/^[a-z0-9_]{3,30}$/, '3–30 letters, numbers or underscores'),
  email: z.string().trim().toLowerCase().max(255).pipe(z.email('Enter a valid email')),
  password: z
    .string()
    .min(8, 'At least 8 characters')
    .refine((value) => new TextEncoder().encode(value).length <= 1024, 'Too long'),
})

export function RegisterForm() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const form = useForm({ resolver: zodResolver(schema) })
  const { errors } = form.formState

  const create = useMutation({
    meta: { requiresAuth: false },
    mutationFn: async (values: z.output<typeof schema>) => {
      await register(values)
      return signIn({ identifier: values.username, password: values.password })
    },
    onSuccess: async () => {
      await acceptSession(queryClient)
      navigate('/', { replace: true })
    },
  })

  return (
    <form
      noValidate
      onSubmit={form.handleSubmit((values) => create.mutate(values))}
      className="grid gap-5"
    >
      <Field
        label="Username"
        autoComplete="username"
        autoCapitalize="none"
        spellCheck={false}
        hint="Lowercase letters, numbers and underscores."
        error={errors.username?.message}
        {...form.register('username')}
      />
      <Field
        label="Email"
        type="email"
        autoComplete="email"
        error={errors.email?.message}
        {...form.register('email')}
      />
      <Field
        label="Password"
        type="password"
        autoComplete="new-password"
        hint="At least 8 characters."
        error={errors.password?.message}
        {...form.register('password')}
      />
      <FormError
        message={
          create.error &&
          (isApiError(create.error, 409)
            ? 'That username or email is taken.'
            : create.error.message)
        }
      />
      <Button type="submit" disabled={create.isPending}>
        {create.isPending ? 'Creating account…' : 'Create account'}
      </Button>
    </form>
  )
}
