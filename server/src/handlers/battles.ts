import type {Request, Response} from "express"
import { createBattle, getBattleById, getBattleByInvite, joinBattle, resetPicks, setPendingPick } from "../db/queries/battles.js"
import { createLobbySchema, joinLobbySchema, pickSongSchema } from "../validation/battles.js"
import { AppError } from "../utils/AppError.js";
import { createRound } from "../db/queries/rounds.js";

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

export async function pickSong(req: Request, res: Response){
    if(!req.userId){
        throw new AppError("Unauthorized", 401)
    }
    const battleId = req.params.battleId as string;
    if(!battleId){
        throw new AppError("Battle not found", 404)
    }
    const {songId} = pickSongSchema.parse(req.body)
    if(!songId){
        throw new AppError("Song not found", 404)
    }
    const battle = await getBattleById(battleId);
    if(!battle){
        throw new AppError("Battle is not currently active", 400)
    }
    const isHost = battle.hostId === req.userId
    const isGuest = battle.guestId === req.userId
    if(!isHost && !isGuest){
        throw new AppError("You are not a participant of this battle", 403)
    }
    if((isHost && battle.hostSongId) || (isGuest && battle.guestSongId)){
        throw new AppError("You already picked a song", 400)
    }
    const field = isHost? "hostSongId" : "guestSongId"
    const updatedBattle = await setPendingPick(battleId , field, songId)
    if(updatedBattle?.hostSongId && updatedBattle?.guestSongId){
        const round = await createRound({battleId, hostSongId: updatedBattle.hostSongId, guestSongId: updatedBattle.guestSongId, roundNumber: battle.currentRound})
        await resetPicks(battleId)
        res.status(201).json({ status: "ROUND_STARTED", round})
        return
    }
    res.status(200).json({status: "WAITING_ON_OPPONENT"})
}
