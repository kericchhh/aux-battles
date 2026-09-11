import { apiFetch } from "./client";
export type Stage = "DRUM" | "BASS" | "MELODY" | "FULL";

export interface BattleRound {
  id: string;
  number: number;
  status: "SONG_PICKS" | "GUESSING" | "FINISHED";
  myStage: Stage;
  myPoints: number;
  myFinished: boolean;
  opponentFinished: boolean;
  myHasPicked: boolean;
  opponentHasPicked: boolean;
  myAttempts: number;
}

export interface BattleView {
  id: string; status: "PENDING" | "ONGOING" | "FINISHED"; inviteCode: string;
  currentRound: number; rounds: number; opponentJoined: boolean;
  outcome: "WIN" | "LOSS" | "DRAW" | null; myScore: number; opponentScore: number;
  round: BattleRound;  
  previousRound: {number: number; myPoints: number; opponentPoints: number} | null;
}

export type GuessResult = {correct: boolean; pointsAwarded: number; playerFinished: boolean; roundFinished: boolean; roundId: string};
export const createBattle = (rounds: number) => apiFetch<{id: string}>("/battles", {method: "POST", body: JSON.stringify({rounds})});
export const joinBattle = (inviteCode: string) => apiFetch<{id: string}>("/battles/join", {method: "POST", body: JSON.stringify({inviteCode})});
export const getBattle = (id: string, signal?: AbortSignal) => apiFetch<BattleView>(`/battles/${id}`, {signal});
export const pickSong = (battleId: string, roundId: string, songId: string) => apiFetch<{status: string}>(`/battles/${battleId}/rounds/picks`, {method: "POST", body: JSON.stringify({roundId, songId})});
export const submitGuess = (roundId: string, guess: string, expectedAttempt: number) => apiFetch<GuessResult>(`/rounds/${roundId}/guess`, {method: "POST", body: JSON.stringify({guess, expectedAttempt})});
