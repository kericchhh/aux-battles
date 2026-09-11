import { db } from "../index.js";
import { usersTable, battlesTable } from "../schema.js";
import { eq, or, sql, and } from "drizzle-orm";

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

export async function getProfile(id: string) {
  const [res] = await db
    .select({
      id: usersTable.id,
      username: usersTable.username,
      avatarUrl: usersTable.avatarUrl,
      createdAt: usersTable.createdAt,
      battlesPlayed: sql<number>`
        count(${battlesTable.id})::int
      `,
      wins: sql<number>`
        count(${battlesTable.id})
        filter (where ${battlesTable.winnerId} = ${usersTable.id})::int
      `,
    })
    .from(usersTable)
    .leftJoin(
      battlesTable,
      and(
        eq(battlesTable.status, "FINISHED"),
        or(
          eq(battlesTable.hostId, usersTable.id),
          eq(battlesTable.guestId, usersTable.id),
        ),
      ),
    )
    .where(eq(usersTable.id, id))
    .groupBy(
      usersTable.id,
      usersTable.username,
      usersTable.avatarUrl,
      usersTable.createdAt,
    );

  return res;
}
