import { apiFetch } from "./client";
import type { BattleView, GuessResult } from "@/lib/types/battle";

export const createBattle = (rounds: number) => apiFetch<{id: string}>("/battles", {method: "POST", body: JSON.stringify({rounds})});
export const joinBattle = (inviteCode: string) => apiFetch<{id: string}>("/battles/join", {method: "POST", body: JSON.stringify({inviteCode})});
export const getBattle = (id: string, signal?: AbortSignal) => apiFetch<BattleView>(`/battles/${id}`, {signal});
export const submitLineup = (battleId: string, songIds: string[]) => apiFetch<{status: "WAITING_ON_OPPONENT" | "BATTLE_STARTED"}>(`/battles/${battleId}/lineup`, {method: "PUT", body: JSON.stringify({songIds})});
export const submitGuess = (roundId: string, guess: string, expectedAttempt: number) => apiFetch<GuessResult>(`/rounds/${roundId}/guess`, {method: "POST", body: JSON.stringify({guess, expectedAttempt})});
