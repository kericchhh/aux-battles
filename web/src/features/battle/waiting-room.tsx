import { Copy } from 'lucide-react'
import { toast } from 'sonner'

import type { Battle } from '@/api/battles'
import { Pending } from '@/components/pending'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

export function WaitingRoom({ battle }: { battle: Battle }) {
  async function copy() {
    try {
      await navigator.clipboard.writeText(battle.inviteCode)
      toast.success('Invite code copied')
    } catch {
      toast.error('Could not copy. Select the code instead.')
    }
  }

  return (
    <Card
      title="Invite an opponent"
      description="Share this code. The battle starts once they join."
      className="max-w-lg"
    >
      <div className="flex flex-wrap items-center gap-3">
        <output className="invite-code" aria-label="Invite code">
          {battle.inviteCode}
        </output>
        <Button variant="secondary" onClick={copy}>
          <Copy aria-hidden />
          Copy code
        </Button>
      </div>
      <Pending>Waiting for someone to join…</Pending>
    </Card>
  )
}
