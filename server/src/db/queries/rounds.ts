import { db } from "../index.js";
import { guessesTable, roundsTable } from "../schema.js";
import { eq, and, sql, count } from "drizzle-orm";
import type { DbExecutor, Transaction } from "../types.js";

type Round = typeof roundsTable.$inferSelect;
type PlayerSide = "host" | "guest";

export async function createRound(data: typeof roundsTable.$inferInsert, tx: Transaction) {
    const [res] = await tx.insert(roundsTable).values(data).returning()
    return res
}

export async function getRoundById(roundId: string, executor: DbExecutor = db) {
    const [res] = await executor.select().from(roundsTable).where(eq(roundsTable.id, roundId))
    return res
}

export async function getRoundByBattleId(battleId: string, executor: DbExecutor = db) {
    const res = await executor.select().from(roundsTable).where(eq(roundsTable.battleId, battleId))
    return res
}

export async function getGuessCount(roundId: string, userId: string, executor: DbExecutor = db): Promise<number> {
    const [res] = await executor
        .select({ value: count() })
        .from(guessesTable)
        .where(
            and(
                eq(guessesTable.roundId, roundId),
                eq(guessesTable.userId, userId),
            ),
        );

    return res?.value ?? 0;
}

export async function makeGuess(data: typeof guessesTable.$inferInsert, executor: DbExecutor = db) {
    const [res] = await executor.insert(guessesTable).values(data).returning()
    return res
}

export async function hasCorrectGuess(roundId: string, userId: string, executor: DbExecutor = db) {
    const [res] = await executor.select().from(guessesTable).where(and(
        eq(guessesTable.roundId, roundId),
        eq(guessesTable.userId, userId),
        eq(guessesTable.correct, true)
    )).limit(1)
    return Boolean(res)
}

export async function getCurrentRoundForUpdate(battleId: string, roundNumber: number, tx: Transaction) {
    const [res] = await tx
        .select()
        .from(roundsTable)
        .where(
            and(
                eq(roundsTable.battleId, battleId),
                eq(roundsTable.roundNumber, roundNumber),
            ),
        )
        .for("update");

    return res;
}

export async function updateRoundSelection(
    roundId: string,
    selection: Pick<Round, "hostSongId" | "guestSongId"> & { status: "SONG_PICKS" | "GUESSING" },
    tx: Transaction
) {
    const [res] = await tx
        .update(roundsTable)
        .set(selection)
        .where(and(
            eq(roundsTable.id, roundId),
            eq(roundsTable.status, "SONG_PICKS")
        )).returning()
    return res
}

export async function updatePlayerProgress(
    roundId: string,
    side: PlayerSide,
    progress: { stage: Round["hostStage"]; points: number; finished: boolean},
    tx: Transaction
) {
    const patch = side === "host"
        ? {hostStage: progress.stage, hostPoints: progress.points, hostFinished: progress.finished}
        : {guestStage: progress.stage, guestPoints: progress.points, guestFinished: progress.finished}
    const finishedColumn = side === "host" ? roundsTable.hostFinished : roundsTable.guestFinished;
    const [res] = await tx
        .update(roundsTable)
        .set(patch)
        .where(and(
            eq(roundsTable.id, roundId),
            eq(roundsTable.status, "GUESSING"),
            eq(finishedColumn, false)
        )).returning()
    return res
}

export async function finishRound(roundId: string, tx: Transaction) {
    const [res] = await tx
        .update(roundsTable)
        .set({status: "FINISHED"})
        .where(and(
            eq(roundsTable.id, roundId),
            eq(roundsTable.status, "GUESSING"),
            eq(roundsTable.hostFinished, true),
            eq(roundsTable.guestFinished, true)
        )).returning()
    return res
}
