import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Flag } from 'lucide-react'
import { toast } from 'sonner'

import { battleKey, forfeitBattle, type Battle } from '@/api/battles'
import { ConfirmButton } from '@/components/confirm-button'

export function ForfeitButton({ battle }: { battle: Battle }) {
  const queryClient = useQueryClient()
  const forfeit = useMutation({
    mutationFn: () => forfeitBattle(battle.id),
    onSuccess: (updated) => queryClient.setQueryData(battleKey(battle.id), updated),
    onError: (error) => {
      toast.error(error.message)
      void queryClient.invalidateQueries({ queryKey: battleKey(battle.id) })
    },
  })
  const copy =
    battle.status === 'PENDING'
      ? {
          title: 'Cancel this lobby?',
          description: 'The invite code stops working and the battle is cancelled.',
          confirm: 'Cancel lobby',
          cancel: 'Keep lobby',
        }
      : {
          title: 'Forfeit this battle?',
          description: 'Your opponent wins the battle. This cannot be undone.',
          confirm: 'Forfeit',
          cancel: 'Keep playing',
        }

  return (
    <ConfirmButton {...copy} disabled={forfeit.isPending} onConfirm={() => forfeit.mutate()}>
      <Flag aria-hidden />
      {copy.confirm}
    </ConfirmButton>
  )
}
