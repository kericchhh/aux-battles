import { apiFetch } from "./client";
export type Stage = "DRUM" | "BASS" | "MELODY" | "FULL";

export interface BattleRound {
  id: string;
  number: number;
  status: "WAITING" | "GUESSING" | "FINISHED";
  myStage: Stage;
  myPoints: number;
  myFinished: boolean;
  opponentFinished: boolean;
  myAttempts: number;
}

export interface BattleView {
  id: string; status: "PENDING" | "SELECTING" | "ONGOING" | "FINISHED"; inviteCode: string;
  currentRound: number; rounds: number; opponentJoined: boolean;
  myLineupLocked: boolean; opponentLineupLocked: boolean;
  outcome: "WIN" | "LOSS" | "DRAW" | null; myScore: number; opponentScore: number;
  round: BattleRound | null;
  previousRound: {number: number; myPoints: number; opponentPoints: number} | null;
}

export type GuessResult = {correct: boolean; pointsAwarded: number; playerFinished: boolean; roundFinished: boolean; roundId: string};
export const createBattle = (rounds: number) => apiFetch<{id: string}>("/battles", {method: "POST", body: JSON.stringify({rounds})});
export const joinBattle = (inviteCode: string) => apiFetch<{id: string}>("/battles/join", {method: "POST", body: JSON.stringify({inviteCode})});
export const getBattle = (id: string, signal?: AbortSignal) => apiFetch<BattleView>(`/battles/${id}`, {signal});
export const submitLineup = (battleId: string, songIds: string[]) => apiFetch<{status: "WAITING_ON_OPPONENT" | "BATTLE_STARTED"}>(`/battles/${battleId}/lineup`, {method: "PUT", body: JSON.stringify({songIds})});
export const submitGuess = (roundId: string, guess: string, expectedAttempt: number) => apiFetch<GuessResult>(`/rounds/${roundId}/guess`, {method: "POST", body: JSON.stringify({guess, expectedAttempt})});
