import { db } from "../index.js";
import { usersTable } from "../schema.js";
import { eq, or } from "drizzle-orm";

export async function registerUserQuery(data: typeof usersTable.$inferInsert) {
    const [res] = await db.insert(usersTable).values(data).returning()
    return res
}

export async function getUserByIdentifier(identifier: string) {
    const [res] = await db.select().from(usersTable).where(or(
        eq(usersTable.email, identifier),
        eq(usersTable.username, identifier)
    ))
    return res
}

export async function getUserById(id: string){
    const [res] = await db.select().from(usersTable).where(eq(usersTable.id, id))
    return res
}
