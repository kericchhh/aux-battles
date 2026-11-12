import { Link } from 'react-router'
import { AuthLayout } from '@/components/layout/auth-layout'
import { RegisterForm } from '@/features/auth/register-form'

export function Register() {
  return (
    <AuthLayout title="Create an account" description="Pick a username to start battling.">
      <RegisterForm />
      <p className="text-dim">
        Have an account?{' '}
        <Link to="/sign-in" className="link">
          Sign in
        </Link>
      </p>
    </AuthLayout>
  )
}
