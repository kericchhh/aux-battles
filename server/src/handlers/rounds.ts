import type { Request, Response } from "express";
import { AppError } from "../utils/AppError.js";
import { submitGuessSchema } from "../validation/rounds.js";
import { addPoints, advanceStage, getGuessCount, getRoundById, makeGuess } from "../db/queries/rounds.js";
import { getBattleById } from "../db/queries/battles.js";
import { getSongById } from "../db/queries/songs.js";

const STAGE_ORDER = ["DRUM", "BASS", "MELODY", "FULL"] as const;
const STAGE_POINTS: Record<string, number> = { DRUM: 100, BASS: 75, MELODY: 50, FULL: 20 }

export async function submitGuess(req: Request, res: Response) {
    if (!req.userId) throw new AppError("Unauthorized", 401)

    const roundId = req.params.roundId as string;
    if(!roundId){
        throw new AppError("Round not found", 404)
    }
    const {guess} = submitGuessSchema.parse(req.body)
    if(!guess){
        throw new AppError("Invalid guess", 404)
    }
    const round = await getRoundById(roundId)
    if(!round){
        throw new AppError("Round not found", 404)
    }
    if(round.status !== "GUESSING"){
        throw new AppError("Round is not accepting guesses", 400)
    }
    const battle = await getBattleById(round.battleId)
    if(!battle){
        throw new AppError("Battle not found", 404)
    }
    const isHost = battle.hostId === req.userId
    const isGuest = battle.guestId === req.userId
    if(!isHost && !isGuest) throw new AppError("You do not participate in this battle", 403)

    const targetSongId = isHost? round.guestSongId : round.hostSongId;
    const stageField = isHost? "hostStage": "guestStage";
    const pointsField = isHost? "hostPoints": "guestPoints";
    const currentStage = round[stageField];

    const targetSong = await getSongById(targetSongId)
    if(!targetSong){
        throw new AppError("Target not found", 404)
    }
    const correct = normalize(guess) === normalize(targetSong.title);
    const attemptNumber = (await getGuessCount(roundId, req.userId)) + 1;
    await makeGuess({ roundId, userId: req.userId, guess, correct, attempt: attemptNumber});
    if(correct){
        const points = STAGE_POINTS[currentStage];
        if(!points){
            throw new AppError("Invalid stage", 500)
        }
        await addPoints(roundId, pointsField, points)
        res.status(200).json({ result: "CORRECT", pointsAwarded: points})
        return
    }
    const currentIndex = STAGE_ORDER.indexOf(currentStage)
    if(currentIndex === STAGE_ORDER.length - 1){
        res.status(200).json({ result: "INCORRECT", stage: currentStage, finalAttempt: true})
        return
    }

    const nextStage = STAGE_ORDER[currentIndex + 1]
    if(!nextStage){
        throw new AppError("Invalid stage", 500)
    }
    await advanceStage(roundId, stageField, nextStage)
    res.status(200).json({result: "INCORRECT", nextStage})
}

function normalize(s: string){
    return s.trim().toLowerCase()
}
