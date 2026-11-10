import { queryOptions } from '@tanstack/react-query'

import { api } from './client'

export type Song = { id: string; title: string; artist: string }

export type SongStatus = Song & {
  status: 'PROCESSING' | 'READY' | 'FAILED'
  processingError: string | null
  workerAvailable: boolean | null
}

export const songsQuery = queryOptions({
  queryKey: ['songs'],
  queryFn: () => api<Song[]>('/songs'),
})

export const songStatusQuery = (id: string) =>
  queryOptions({
    queryKey: ['songs', id, 'status'],
    queryFn: () => api<SongStatus>(`/songs/${encodeURIComponent(id)}/status`),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'READY' || status === 'FAILED' ? false : 3000
    },
  })

export const uploadSong = (form: FormData) =>
  api<{ id: string; status: string }>('/songs', { method: 'POST', body: form })
