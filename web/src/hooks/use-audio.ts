import { useCallback, useEffect, useRef, useState } from 'react'

export type AudioState = { playing: boolean; failed: boolean; time: number; duration: number }

const idle: AudioState = { playing: false, failed: false, time: 0, duration: 0 }

export function useAudio(src: string | null, autoPlay = false) {
  const element = useRef<HTMLAudioElement | null>(null)
  const auto = useRef(autoPlay)
  const [state, setState] = useState(idle)
  const [source, setSource] = useState({ src })
  if (source.src !== src) setSource({ src })

  useEffect(() => {
    auto.current = autoPlay
  }, [autoPlay])

  useEffect(() => {
    if (!source.src) return
    const audio = new Audio(source.src)
    audio.preload = 'auto'
    element.current = audio
    const sync = () =>
      setState({
        playing: !audio.paused && !audio.ended,
        failed: audio.error !== null,
        time: audio.currentTime,
        duration: Number.isFinite(audio.duration) ? audio.duration : 0,
      })
    const events = ['play', 'pause', 'ended', 'timeupdate', 'loadedmetadata', 'error'] as const
    for (const event of events) audio.addEventListener(event, sync)
    if (auto.current) void audio.play().catch(() => {})
    return () => {
      for (const event of events) audio.removeEventListener(event, sync)
      audio.pause()
      audio.removeAttribute('src')
      audio.load()
      element.current = null
      setState(idle)
    }
  }, [source])

  const toggle = useCallback(() => {
    const audio = element.current
    if (!audio) return
    if (audio.paused || audio.ended) void audio.play().catch(() => {})
    else audio.pause()
  }, [])

  const seek = useCallback((time: number) => {
    if (element.current) element.current.currentTime = time
  }, [])

  const retry = useCallback(() => setSource({ src }), [src])

  return { ...state, toggle, seek, retry }
}
