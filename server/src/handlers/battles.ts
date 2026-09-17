import type { Request, Response } from "express";
import * as game from "../services/game.js";
import { battleParams, createBattleInput, joinBattleInput, lineupInput } from "../validation/game.js";
import { AppError } from "../utils/AppError.js";
import { notifyBattleChanged } from "../services/game-events.js";

function userId(req: Request) {
  if (!req.userId) throw new AppError("Unauthorized", 401);
  return req.userId;
}

export async function createLobby(req: Request, res: Response) {
  const { rounds } = createBattleInput.parse(req.body);
  res.status(201).json(await game.createBattle(userId(req), rounds));
}

export async function joinLobby(req: Request, res: Response) {
  const { inviteCode } = joinBattleInput.parse(req.body);
  const result = await game.joinBattle(inviteCode, userId(req));
  notifyBattleChanged(result.id);
  res.json(result);
}

export async function getLobby(req: Request, res: Response) {
  const { battleId } = battleParams.parse(req.params);
  res.setHeader("Cache-Control", "no-store");
  res.json(await game.getBattleView(battleId, userId(req)));
}

export const getBattleByIdHandler = getLobby;

export async function submitLineup(req: Request, res: Response) {
  const { battleId } = battleParams.parse(req.params);
  const input = lineupInput.parse(req.body);
  const result = await game.submitLineup(battleId, userId(req), input.songIds);
  notifyBattleChanged(battleId);
  res.json(result);
}
