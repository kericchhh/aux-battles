import { isActive, type BattleStatus, type Outcome } from '@/api/battles'
import { Badge } from '@/components/ui/badge'

const labels: Record<BattleStatus, string> = {
  PENDING: 'Lobby',
  SELECTING: 'Picking songs',
  ONGOING: 'Playing',
  FINISHED: 'Finished',
  CANCELLED: 'Cancelled',
}

const outcomeLabels: Record<Outcome, string> = { WIN: 'Won', LOSS: 'Lost', DRAW: 'Draw' }

export function StatusBadge({
  status,
  outcome,
}: {
  status: BattleStatus
  outcome: Outcome | null
}) {
  return (
    <Badge tone={outcome === 'WIN' ? 'win' : isActive(status) ? 'live' : 'idle'}>
      {outcome ? outcomeLabels[outcome] : labels[status]}
    </Badge>
  )
}
