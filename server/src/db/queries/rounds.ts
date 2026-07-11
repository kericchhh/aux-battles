import { db } from "../index.js";
import { guessesTable, roundsTable } from "../schema.js";
import { eq, and, sql } from "drizzle-orm";

export async function createRound(data: Omit<typeof roundsTable.$inferInsert, "status">){
    const [res] = await db.insert(roundsTable).values({...data, status: "GUESSING"}).returning()
    return res
}

export async function getRoundById(roundId: string) {
    const [res] = await db.select().from(roundsTable).where(eq(roundsTable.id, roundId))
    return res
}

export async function getGuessCount(roundId: string, userId: string){
    const res = await db.select().from(guessesTable).where(and(eq(guessesTable.roundId, roundId), eq(guessesTable.userId, userId)))
    return res.length
}

export async function makeGuess(data: typeof guessesTable.$inferInsert){
    const [res] = await db.insert(guessesTable).values(data).returning()
    return res
}

export async function advanceStage(roundId: string, field: "hostStage" | "guestStage", nextStage: string){
    const [res] = await db.update(roundsTable).set({ [field] : nextStage}).where(eq(roundsTable.id, roundId)).returning()
    return res
}

export async function addPoints(roundId: string, field: "hostPoints" | "guestPoints", points: number) {
    const [res] = await db.update(roundsTable).set({ [field]: sql`${roundsTable[field]} + ${points}`}).where(eq(roundsTable.id, roundId)).returning()
    return res
}
