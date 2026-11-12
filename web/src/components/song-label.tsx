type Song = { title: string; artist: string }

export function SongLabel({ song }: { song: Song }) {
  return (
    <span className="grid min-w-0 flex-1">
      <span className="truncate font-medium">{song.title}</span>
      <span className="truncate text-sm text-dim">{song.artist}</span>
    </span>
  )
}

export function SongCredit({ song }: { song: Song }) {
  return (
    <>
      {song.title} <span className="text-dim">by {song.artist}</span>
    </>
  )
}
