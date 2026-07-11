import type {Request, Response} from "express"
import { createBattle, getBattleByInvite, joinBattle } from "../db/queries/battles.js"
import { createLobbySchema, joinLobbySchema } from "../validation/battles.js"
import { AppError } from "../utils/AppError.js";

export async function createLobby(req: Request, res: Response) {
    if(!req.userId){
        throw new AppError("Unauthorized", 401)
    }
    const {rounds} = createLobbySchema.parse(req.body);
    const battle = await createBattle({hostId: req.userId, rounds});
    res.status(201).json(battle)
}

export async function joinLobby(req: Request, res: Response){
    if(!req.userId) {
        throw new AppError("Unauthorized", 401)
    }
    const {inviteCode} = joinLobbySchema.parse(req.body)
    const battle = await getBattleByInvite(inviteCode)
    if(!battle){
        throw new AppError("Battle not found", 404)
    }
    if(battle.status !== "PENDING"){
        throw new AppError("Battle has already started", 400)
    }
    if(battle.hostId === req.userId){
        throw new AppError("You can't join your own battle", 400)
    }
    if(battle.guestId){
        throw new AppError("Lobby is full", 400)
    }
    const updatedBattle = await joinBattle(battle.id, req.userId)
    res.status(200).json(updatedBattle)
}
