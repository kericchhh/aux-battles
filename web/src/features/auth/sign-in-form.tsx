import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useLocation, useNavigate } from 'react-router'
import { z } from 'zod'
import { returnPath } from './return-path'

import { acceptSession, signIn } from '@/api/auth'
import { isApiError } from '@/api/client'
import { Field } from '@/components/field'
import { FormError } from '@/components/form-error'
import { Button } from '@/components/ui/button'

const schema = z.object({
  identifier: z.string().trim().min(1, 'Enter your username or email'),
  password: z.string().min(1, 'Enter your password'),
})

export function SignInForm() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const from = returnPath(useLocation().state)
  const form = useForm({ resolver: zodResolver(schema) })
  const { errors } = form.formState

  const login = useMutation({
    meta: { requiresAuth: false },
    mutationFn: signIn,
    onSuccess: async () => {
      await acceptSession(queryClient)
      navigate(from, { replace: true })
    },
  })

  return (
    <form
      noValidate
      onSubmit={form.handleSubmit((values) => login.mutate(values))}
      className="grid gap-5"
    >
      <Field
        label="Username or email"
        autoComplete="username"
        error={errors.identifier?.message}
        {...form.register('identifier')}
      />
      <Field
        label="Password"
        type="password"
        autoComplete="current-password"
        error={errors.password?.message}
        {...form.register('password')}
      />
      <FormError
        message={
          login.error &&
          (isApiError(login.error, 401)
            ? 'Wrong username, email or password.'
            : login.error.message)
        }
      />
      <Button type="submit" disabled={login.isPending}>
        {login.isPending ? 'Signing in…' : 'Sign in'}
      </Button>
    </form>
  )
}
