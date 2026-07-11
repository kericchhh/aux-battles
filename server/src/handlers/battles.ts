import type {Request, Response} from "express"
import { createBattle } from "../db/queries/battles.js"
import { createLobbySchema } from "../validation/battles.js"
import { AppError } from "../utils/AppError.js";

export async function createLobby(req: Request, res: Response) {
    if(!req.userId){
        throw new AppError("Unauthorized", 404)
    }
    const {rounds} = createLobbySchema.parse(req.body);
    const battle = await createBattle({hostId: req.userId, rounds});
    res.status(201).json(battle)
}
