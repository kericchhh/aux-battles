import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as api from "@/api/battles";
import { queryKeys } from "@/lib/queryKeys";
export function useBattle(battleId: string, userId: string) {
  const cache = useQueryClient();
  const queryKey = queryKeys.battle(userId, battleId);
  const query = useQuery({queryKey, queryFn: ({signal}) => api.getBattle(battleId, signal),
    refetchInterval: q => q.state.data?.status === "FINISHED" ? false : 10000});
  const refresh = () => cache.invalidateQueries({queryKey});
  const lineup = useMutation({mutationFn: (songIds: string[]) => api.submitLineup(battleId, songIds), onSettled: refresh});
  const guess = useMutation({mutationFn: (input: {roundId: string; guess: string; expectedAttempt: number}) => api.submitGuess(input.roundId, input.guess, input.expectedAttempt), onSettled: refresh});
  return {...query, lineup, guess};
}
export type BattleGame = ReturnType<typeof useBattle>;
