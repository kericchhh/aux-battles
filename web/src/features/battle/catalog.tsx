import { useInfiniteQuery } from '@tanstack/react-query'
import { Check, Pause, Play, Plus } from 'lucide-react'
import { useState } from 'react'

import { songPreviewUrl, songsQuery, type Song } from '@/api/songs'
import { DataState } from '@/components/data-state'
import { Field } from '@/components/field'
import { SongLabel } from '@/components/song-label'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { useAudioPreview } from '@/hooks/use-audio-preview'
import { useDebounced } from '@/hooks/use-debounced'
import { countLabel } from '@/lib/format'

type CatalogProps = {
  rounds: number
  picked: Song[]
  full: boolean
  onAdd: (song: Song) => void
}

export function Catalog({ rounds, picked, full, onAdd }: CatalogProps) {
  const [search, setSearch] = useState('')
  const query = useDebounced(search.trim(), 250)
  const songs = useInfiniteQuery(songsQuery(query))
  const preview = useAudioPreview(songPreviewUrl)
  const list = songs.data?.pages.flatMap((page) => page.songs) ?? []
  const searching = search.trim() !== query || songs.isPlaceholderData || songs.isPending
  const pickedIds = new Set(picked.map((song) => song.id))

  return (
    <Card title="Catalog" description={`Pick ${countLabel(rounds, 'song')}, one per round.`}>
      <Field
        label="Search songs"
        type="search"
        value={search}
        onChange={(event) => setSearch(event.target.value)}
        maxLength={100}
        autoComplete="off"
        placeholder="Title or artist"
      />
      <p role="status" className="sr-only">
        {!searching && songs.isSuccess && `${countLabel(list.length, 'song')} shown`}
      </p>
      <div role="status" className="live">
        {preview.failed && <p className="feedback">That preview could not be played.</p>}
      </div>
      <DataState
        query={songs}
        loading="Loading songs…"
        error="Could not load the catalog."
        empty={
          list.length === 0 && (
            <p className="text-dim">
              {searching
                ? 'Searching songs…'
                : query
                  ? `No songs match “${query}”.`
                  : 'The catalog is empty. Ask an admin to upload songs.'}
            </p>
          )
        }
      >
        <ul className="grid gap-1">
          {list.map((song) => {
            const playing = preview.playing === song.id
            const isPicked = pickedIds.has(song.id)
            const unavailable = full || isPicked
            return (
              <li key={song.id} className="song-row">
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={`${playing ? 'Stop' : 'Preview'} ${song.title}`}
                  aria-pressed={playing}
                  onClick={() => preview.toggle(song.id)}
                >
                  {playing ? <Pause aria-hidden /> : <Play aria-hidden />}
                </Button>
                <SongLabel song={song} />
                <Button
                  variant="secondary"
                  size="icon"
                  aria-label={`${isPicked ? 'Added' : 'Add'} ${song.title}`}
                  aria-disabled={unavailable}
                  onClick={() => unavailable || onAdd(song)}
                >
                  {isPicked ? <Check aria-hidden /> : <Plus aria-hidden />}
                </Button>
              </li>
            )
          })}
        </ul>
      </DataState>
      {songs.hasNextPage && !searching && (
        <Button
          variant="secondary"
          disabled={songs.isFetchingNextPage}
          onClick={() => songs.fetchNextPage()}
        >
          {songs.isFetchingNextPage ? 'Loading…' : 'Load more'}
        </Button>
      )}
    </Card>
  )
}
