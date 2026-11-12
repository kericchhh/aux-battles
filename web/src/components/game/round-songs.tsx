import type { ReactNode } from 'react'

import type { CompletedRound } from '@/api/battles'
import { SongCredit } from '@/components/song-label'
import { cn } from '@/lib/utils'

export function RoundSongs({
  round,
  className,
  children,
}: {
  round: Pick<CompletedRound, 'mySong' | 'opponentSong'>
  className?: string
  children?: ReactNode
}) {
  return (
    <dl className={cn('facts', className)}>
      <div>
        <dt>Their song</dt>
        <dd>
          <SongCredit song={round.opponentSong} />
        </dd>
      </div>
      <div>
        <dt>Your song</dt>
        <dd>
          <SongCredit song={round.mySong} />
        </dd>
      </div>
      {children}
    </dl>
  )
}
