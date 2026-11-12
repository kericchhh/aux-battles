import { useState } from 'react'

import { useAudio } from '@/hooks/use-audio'

export function useAudioPreview(urlFor: (id: string) => string) {
  const [id, setId] = useState<string | null>(null)
  const audio = useAudio(id ? urlFor(id) : null, true)
  const ended = id !== null && audio.duration > 0 && audio.time >= audio.duration
  const stopped = ended || audio.failed

  function toggle(next: string) {
    if (id !== next) return setId(next)
    if (audio.failed) return audio.retry()
    if (!ended) return setId(null)
    audio.seek(0)
    audio.toggle()
  }

  return { playing: stopped ? null : id, failed: audio.failed ? id : null, toggle }
}
