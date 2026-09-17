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

export interface GuessResult {
  correct: boolean;
  pointsAwarded: number;
  playerFinished: boolean;
  roundFinished: boolean;
  roundId: string;
}
