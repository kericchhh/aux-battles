import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, X } from 'lucide-react'
import { useLayoutEffect, useRef, useState } from 'react'

import { battleKey, lockLineup, opponentName, type Battle } from '@/api/battles'
import type { Song } from '@/api/songs'
import { FormError } from '@/components/form-error'
import { Pending } from '@/components/pending'
import { SongLabel } from '@/components/song-label'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

import { Catalog } from './catalog'

export function LineupPicker({ battle }: { battle: Battle }) {
  if (!battle.myLineupLocked) return <LineupForm battle={battle} />
  return (
    <Card title="Lineup locked" className="max-w-lg">
      <Pending>
        {battle.opponentLineupLocked
          ? 'Starting the first round…'
          : `Waiting for ${opponentName(battle)} to lock their lineup…`}
      </Pending>
    </Card>
  )
}

function LineupForm({ battle }: { battle: Battle }) {
  const queryClient = useQueryClient()
  const [lineup, setLineup] = useState<Song[]>([])
  const [announcement, setAnnouncement] = useState('')
  const slots = useRef<HTMLOListElement>(null)
  const movedButton = useRef<HTMLButtonElement | null>(null)
  useLayoutEffect(() => {
    movedButton.current?.focus()
    movedButton.current = null
  })
  const full = lineup.length === battle.rounds
  const lock = useMutation({
    mutationFn: () =>
      lockLineup(
        battle.id,
        lineup.map((song) => song.id),
      ),
    onSettled: () => queryClient.invalidateQueries({ queryKey: battleKey(battle.id) }),
  })

  function add(song: Song) {
    setLineup([...lineup, song])
    setAnnouncement(`${song.title} added as round ${lineup.length + 1}`)
  }
  function remove(index: number) {
    setLineup(lineup.filter((_, i) => i !== index))
    setAnnouncement(`${lineup[index]!.title} removed`)
    slots.current?.focus()
  }
  function move(index: number, by: number, button: HTMLButtonElement) {
    movedButton.current = button
    const next = [...lineup]
    ;[next[index], next[index + by]] = [next[index + by]!, next[index]!]
    setLineup(next)
    setAnnouncement(`${lineup[index]!.title} moved to round ${index + by + 1}`)
  }

  return (
    <div className="grid items-start gap-5 lg:grid-cols-2">
      <Catalog rounds={battle.rounds} picked={lineup} full={full} onAdd={add} />
      <Card
        title="Your lineup"
        description="Your opponent guesses these in order. You cannot change them once locked."
      >
        <ol ref={slots} tabIndex={-1} aria-label="Lineup" className="grid gap-1">
          {Array.from({ length: battle.rounds }, (_, i) => {
            const song = lineup[i]
            return (
              <li
                key={song?.id ?? `empty-${i}`}
                className="lineup-slot"
                data-empty={!song || undefined}
              >
                <span className="font-mono text-dim tabular">{i + 1}</span>
                {song ? (
                  <>
                    <SongLabel song={song} />
                    <SlotActions
                      song={song}
                      first={i === 0}
                      last={i === lineup.length - 1}
                      onMove={(by, button) => move(i, by, button)}
                      onRemove={() => remove(i)}
                    />
                  </>
                ) : (
                  <span className="text-dim">Pick a song</span>
                )}
              </li>
            )
          })}
        </ol>
        <p role="status" className="sr-only">
          {announcement}
        </p>
        <FormError message={lock.error?.message} />
        <Button disabled={!full || lock.isPending} onClick={() => lock.mutate()}>
          {lock.isPending
            ? 'Locking…'
            : full
              ? 'Lock lineup'
              : `Pick ${battle.rounds - lineup.length} more`}
        </Button>
      </Card>
    </div>
  )
}

type SlotActionsProps = {
  song: Song
  first: boolean
  last: boolean
  onMove: (by: number, button: HTMLButtonElement) => void
  onRemove: () => void
}

function SlotActions({ song, first, last, onMove, onRemove }: SlotActionsProps) {
  return (
    <span className="flex">
      <Button
        variant="ghost"
        size="icon"
        aria-label={`Move ${song.title} up`}
        aria-disabled={first}
        onClick={(event) => first || onMove(-1, event.currentTarget)}
      >
        <ArrowUp aria-hidden />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        aria-label={`Move ${song.title} down`}
        aria-disabled={last}
        onClick={(event) => last || onMove(1, event.currentTarget)}
      >
        <ArrowDown aria-hidden />
      </Button>
      <Button variant="ghost" size="icon" aria-label={`Remove ${song.title}`} onClick={onRemove}>
        <X aria-hidden />
      </Button>
    </span>
  )
}
