import { useMutation, useQueryClient } from '@tanstack/react-query'
import { LogOut, Upload } from 'lucide-react'
import { Link, Outlet, useNavigate } from 'react-router'
import { toast } from 'sonner'

import { signOut, type User } from '@/api/auth'
import { HowToPlay } from '@/components/game/how-to-play'
import { Brand } from '@/components/layout/brand'
import { Button } from '@/components/ui/button'

export function AppShell({ user }: { user: User }) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const logout = useMutation({
    mutationFn: signOut,
    onSuccess: () => {
      queryClient.clear()
      navigate('/sign-in', { replace: true })
    },
    onError: () => toast.error('Could not sign out. Try again.'),
  })

  return (
    <div className="flex min-h-dvh flex-col">
      <a href="#main" className="skip-link">
        Skip to content
      </a>
      <header className="topbar">
        <div className="shell flex h-16 items-center gap-4">
          <Brand />
          <div className="ml-auto flex items-center gap-2">
            {user.role === 'ADMIN' && (
              <Button variant="ghost" asChild>
                <Link to="/upload">
                  <Upload aria-hidden />
                  <span className="hidden sm:inline">Upload</span>
                  <span className="sr-only sm:hidden">Upload a song</span>
                </Link>
              </Button>
            )}
            <HowToPlay />
            <span className="avatar" aria-hidden>
              {user.username.slice(0, 1)}
            </span>
            <span className="hidden sm:inline">{user.username}</span>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Sign out"
              title="Sign out"
              data-nudge
              onClick={() => logout.mutate()}
              disabled={logout.isPending}
            >
              <LogOut aria-hidden />
            </Button>
          </div>
        </div>
      </header>
      <main id="main" tabIndex={-1} className="shell flex-1 py-8 outline-none sm:py-12">
        <Outlet />
      </main>
    </div>
  )
}
