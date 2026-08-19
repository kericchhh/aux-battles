import type {Request, Response} from "express"
import { createBattle, getBattleByInvite, joinBattle, pickSongTransaction, getBattleById} from "../db/queries/battles.js"
import { createLobbySchema, joinLobbySchema, pickSongSchema, battleIdSchema } from "../validation/battles.js"
import { AppError } from "../utils/AppError.js";
import { getSongById } from "../db/queries/songs.js";
import { getIO } from "../socket.js";
import { getRoundByBattleId, getRoundById } from "../db/queries/rounds.js";

export async function getLobby(req: Request, res: Response) {
    if(!req.userId){
        throw new AppError("Unauthorized", 401)
    }
    const parsedParams = battleIdSchema.safeParse(req.params)
    if(!parsedParams.success){
        throw new AppError("Invalid battle id", 400)
    }
    const {battleId} = parsedParams.data
    const battle = await getBattleById(battleId)
    if(!battle){
        throw new AppError("Battle not found", 404)
    }
    const isParticipant = battle.hostId === req.userId || battle.guestId === req.userId
    if(!isParticipant) {
        throw new AppError("You are not a participant of this battle", 403)
    }

    const rounds = await getRoundById(battleId)
    const round = rounds.find((item) => item.roundNumber === battle.currentRound) ?? null
    res.status(200).json({battle, round})
}

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

    const song = await getSongById(songId)
    if(!song) throw new AppError("Song not found", 404)
    if(song.status !== "READY") throw new AppError("Song is not available yet", 400)
    const result = await pickSongTransaction(battleId, req.userId, songId)
    getIO().to(battleId).emit("battle:update", result)
    res.status(result.status === "ROUND_STARTED" ? 201 : 200).json(result)
}
