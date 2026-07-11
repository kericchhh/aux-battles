import { eq } from "drizzle-orm";
import { db } from "../index.js";
import { battlesTable } from "../schema.js";
import { customAlphabet } from "nanoid";

const generateInviteCode = customAlphabet(
    "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
    6
);

export async function createBattle(data: Omit<typeof battlesTable.$inferInsert, "inviteCode">) {
    const inviteCode = generateInviteCode()
    const [res] = await db.insert(battlesTable).values({...data, inviteCode}).returning();
    return res
}

export async function getBattleByInvite(code: string){
    const [res] = await db.select().from(battlesTable).where(eq(battlesTable.inviteCode, code))
    return res
}

export async function joinBattle(battleId:string, guestId: string){
    const [res] = await db.update(battlesTable).set({ guestId, status: "ONGOING"}).where(eq(battlesTable.id, battleId)).returning()
    return res
}
