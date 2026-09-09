import { z } from "zod";

export const battleParams = z.object({ battleId: z.string().uuid() });
export const roundParams = z.object({ roundId: z.string().uuid() });
export const createBattleInput = z.object({ rounds: z.number().int().min(1).max(10).default(5) });
export const joinBattleInput = z.object({
  inviteCode: z.string().trim().toUpperCase().regex(/^[A-Z0-9]{6}$/),
});
export const pickSongInput = z.object({ roundId: z.string().uuid(), songId: z.string().uuid() });
export const guessInput = z.object({
  guess: z.string().trim().min(1).max(255),
  expectedAttempt: z.number().int().min(1).max(4),
});
