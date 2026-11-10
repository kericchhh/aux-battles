import type { Stage } from '@/api/battles'

export const stages: { id: Stage; label: string; points: number }[] = [
  { id: 'DRUM', label: 'Drums', points: 100 },
  { id: 'BASS', label: 'Bass', points: 75 },
  { id: 'MELODY', label: 'Melody', points: 50 },
  { id: 'FULL', label: 'Full', points: 20 },
]
