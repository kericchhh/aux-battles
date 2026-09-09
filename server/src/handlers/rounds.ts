import path from "node:path";
import type { Request, Response, NextFunction } from "express";
import * as game from "../services/game.js";
import { roundParams, guessInput } from "../validation/game.js";
import { AppError } from "../utils/AppError.js";
import { notifyBattleChanged } from "../services/game-events.js";

export async function submitGuess(req: Request, res: Response) {
  if (!req.userId) throw new AppError("Unauthorized", 401);
  const { roundId } = roundParams.parse(req.params);
  const input = guessInput.parse(req.body);
  const result = await game.submitGuess(roundId, req.userId, input.guess, input.expectedAttempt);
  notifyBattleChanged(result.battleId);
  res.json(result);
}

export async function getRoundAudio(req: Request, res: Response, next: NextFunction) {
  if (!req.userId) throw new AppError("Unauthorized", 401);
  const { roundId } = roundParams.parse(req.params);
  const audioPath = await game.getAudioPath(roundId, req.userId);
  res.setHeader("Cache-Control", "private, no-store");
  res.type("audio/mpeg");
  res.sendFile(path.resolve(audioPath), (error) => { if (error) next(error); });
}
