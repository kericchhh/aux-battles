import { infiniteQueryOptions, keepPreviousData, queryOptions } from '@tanstack/react-query'

import { api } from './client'

export type Song = { id: string; title: string; artist: string }
export type SongPage = { songs: Song[]; nextCursor: string | null }

export type SongStatus = Song & {
  status: 'PROCESSING' | 'READY' | 'FAILED'
  processingError: string | null
  workerAvailable: boolean | null
}

export const songPageSize = 20

export const songsQuery = (search: string) =>
  infiniteQueryOptions({
    queryKey: ['songs', 'list', search],
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: String(songPageSize) })
      if (search) params.set('q', search)
      if (pageParam) params.set('cursor', pageParam)
      return api<SongPage>(`/songs?${params}`, { signal })
    },
    initialPageParam: '',
    getNextPageParam: (page) => page.nextCursor ?? undefined,
    placeholderData: keepPreviousData,
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

export const songPreviewUrl = (id: string) => `/api/songs/${encodeURIComponent(id)}/audio/full`

export const uploadSong = (form: FormData) =>
  api<{ id: string; status: string }>('/songs', { method: 'POST', body: form })
