import { eq, and, isNull, ne } from "drizzle-orm";
import { db } from "../index.js";
import { battlesTable, roundsTable } from "../schema.js";
import { customAlphabet } from "nanoid";
import { getRoundByBattleId } from "./rounds.js";
import { AppError } from "../../utils/AppError.js";
import type { DbExecutor, Transaction } from "../types.js";

const generateInviteCode = customAlphabet(
    "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
    6
);

export async function createBattle(data: Pick<typeof battlesTable.$inferInsert, "hostId" | "rounds">, executor: DbExecutor = db) {
    const [res] = await executor
        .insert(battlesTable)
        .values({ ...data, inviteCode: generateInviteCode() })
        .onConflictDoNothing({ target: battlesTable.inviteCode }).returning();
    return res
}

export async function getBattleByInvite(code: string, executor: DbExecutor = db) {
    const [res] = await executor.select().from(battlesTable).where(eq(battlesTable.inviteCode, code))
    return res
}

export async function getBattleById(battleId: string, executor: DbExecutor = db) {
    const [res] = await executor.select().from(battlesTable).where(eq(battlesTable.id, battleId))
    return res
}

export async function joinBattle(battleId: string, guestId: string, tx: Transaction) {
    const [res] = await tx
        .update(battlesTable)
        .set({ guestId, status: "ONGOING" })
        .where(and(
            eq(battlesTable.id, battleId),
            eq(battlesTable.status, "PENDING"),
            isNull(battlesTable.guestId),
            ne(battlesTable.hostId, guestId)
        )).returning()
    return res
}

export async function advanceBattle(battleId: string, currentRound: number, tx: Transaction) {
    const [res] = await tx
        .update(battlesTable)
        .set({currentRound: currentRound + 1})
        .where(and(
            eq(battlesTable.id, battleId),
            eq(battlesTable.status, "ONGOING"),
            eq(battlesTable.currentRound, currentRound)
        )).returning()
    return res
}

export async function getBattleForUpdate(battleId: string, tx: Transaction,) {
    const [res] = await tx.select().from(battlesTable).where(eq(battlesTable.id, battleId)).for("update");

    return res;
}

export async function finishBattle(battleId: string, winnerId: string | null, tx: Transaction) {
    const [res] = await tx
        .update(battlesTable)
        .set({ status: "FINISHED", winnerId })
        .where(and(
            eq(battlesTable.id, battleId),
            eq(battlesTable.status, "ONGOING")
        )).returning()
    return res
}
