import { useQuery } from '@tanstack/react-query'

import { songStatusQuery, type SongStatus } from '@/api/songs'
import { SongLabel } from '@/components/song-label'
import { Badge } from '@/components/ui/badge'

function describe(song: SongStatus | undefined, failed: boolean) {
  if (!song) return { label: failed ? 'Status unavailable' : 'Checking…', tone: 'live' } as const
  if (song.status === 'READY') return { label: 'Ready', tone: 'win' } as const
  if (song.status === 'FAILED') return { label: 'Failed', tone: 'alarm' } as const
  if (song.workerAvailable === false)
    return { label: 'Queued, no worker running', tone: 'live' } as const
  return { label: 'Processing', tone: 'live' } as const
}

export function UploadStatus({ id }: { id: string }) {
  const status = useQuery(songStatusQuery(id))
  const song = status.data
  const { label, tone } = describe(song, status.isError)

  return (
    <li className="song-row">
      <SongLabel
        song={{
          title: song?.title ?? 'New upload',
          artist: song?.processingError ?? song?.artist ?? '',
        }}
      />
      <Badge role="status" tone={tone}>
        {label}
      </Badge>
    </li>
  )
}
