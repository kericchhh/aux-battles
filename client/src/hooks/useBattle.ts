import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as api from "../api/battles";
export function useBattle(battleId: string, userId: string) {
  const cache = useQueryClient();
  const queryKey = ["battle", userId, battleId];
  const query = useQuery({queryKey, queryFn: ({signal}) => api.getBattle(battleId, signal),
    refetchInterval: q => q.state.data?.status === "FINISHED" ? false : 10000});
  const refresh = () => cache.invalidateQueries({queryKey});
  const pick = useMutation({mutationFn: (input: {roundId: string; songId: string}) => api.pickSong(battleId, input.roundId, input.songId), onSettled: refresh});
  const guess = useMutation({mutationFn: (input: {roundId: string; guess: string; expectedAttempt: number}) => api.submitGuess(input.roundId, input.guess, input.expectedAttempt), onSettled: refresh});
  return {...query, pick, guess};
}
export type BattleGame = ReturnType<typeof useBattle>;
