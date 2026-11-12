import { Link } from 'react-router'
import { AuthLayout } from '@/components/layout/auth-layout'
import { SignInForm } from '@/features/auth/sign-in-form'

export function SignIn() {
  return (
    <AuthLayout title="Sign in" description="Welcome back to aux/battles.">
      <SignInForm />
      <p className="text-dim">
        New here?{' '}
        <Link to="/register" className="link">
          Create an account
        </Link>
      </p>
    </AuthLayout>
  )
}
