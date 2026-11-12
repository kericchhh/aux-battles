import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'

import { battleEventsUrl, battleKey, battleQuery, isActive } from '@/api/battles'

const pollMs = 3000
const livePollMs = 30_000

export function useLiveBattle(battleId: string) {
  const queryClient = useQueryClient()
  const [live, setLive] = useState(false)
  const battle = useQuery({
    ...battleQuery(battleId),
    refetchInterval: (query) => {
      const data = query.state.data
      if (data && !isActive(data.status)) return false
      return live ? livePollMs : pollMs
    },
  })
  const active = !battle.data || isActive(battle.data.status)

  useEffect(() => {
    if (!active || typeof EventSource === 'undefined') return
    const source = new EventSource(battleEventsUrl(battleId))
    source.addEventListener('changed', () => {
      setLive(true)
      void queryClient.invalidateQueries({ queryKey: battleKey(battleId) })
    })
    source.onerror = () => setLive(source.readyState === EventSource.OPEN)
    return () => {
      source.close()
      setLive(false)
    }
  }, [active, battleId, queryClient])

  return { battle }
}
