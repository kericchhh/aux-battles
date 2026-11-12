import { Pause, Play } from 'lucide-react'
import type { CSSProperties } from 'react'

import { Button } from '@/components/ui/button'
import type { useAudio } from '@/hooks/use-audio'

const clock = (seconds: number) =>
  `${Math.floor(seconds / 60)}:${String(Math.floor(seconds % 60)).padStart(2, '0')}`

export function AudioPlayer({
  label,
  audio,
}: {
  label: string
  audio: ReturnType<typeof useAudio>
}) {
  const progress = audio.duration ? audio.time / audio.duration : 0

  if (audio.failed) {
    return (
      <p role="alert" className="text-alarm">
        This audio could not be loaded.
      </p>
    )
  }

  return (
    <div role="group" aria-label={label} className="player">
      <Button size="icon" aria-label={audio.playing ? 'Pause' : 'Play'} onClick={audio.toggle}>
        {audio.playing ? <Pause aria-hidden /> : <Play aria-hidden />}
      </Button>
      <input
        type="range"
        aria-label="Seek"
        aria-valuetext={`${clock(audio.time)} of ${clock(audio.duration)}`}
        min={0}
        max={audio.duration || 1}
        step={1}
        value={audio.time}
        onChange={(event) => audio.seek(Number(event.target.value))}
        className="player-seek"
        style={{ '--progress': progress } as CSSProperties}
      />
      <span className="player-time">
        {clock(audio.time)} / {clock(audio.duration)}
      </span>
    </div>
  )
}
