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
