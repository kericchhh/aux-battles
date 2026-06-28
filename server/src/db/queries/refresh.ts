import { eq, and, isNull, gt } from "drizzle-orm"
import { db } from "../index.js"
import { refreshTokens, usersTable } from "../schema.js"


export async function saveRefreshToken(userId: string, token: string, expiresAt: Date) {
    const res = await db.insert(refreshTokens).values({ userId, token, expiresAt }).returning()
    return res.length > 0
}


export async function userForRefreshToken(token: string) {
    const [res] = await db
        .select({ user: usersTable })
        .from(usersTable)
        .innerJoin(refreshTokens, eq(usersTable.id, refreshTokens.userId))
        .where(
            and(
                eq(refreshTokens.token, token),
                isNull(refreshTokens.revokedAt),
                gt(refreshTokens.expiresAt, new Date())
            )
        )
        .limit(1)

    return res
}

export async function revokeRefreshToken(token: string) {
    const data = await db.update(refreshTokens).set({ revokedAt: new Date() }).where(eq(refreshTokens.token, token)).returning()
    if (data.length === 0) {
        throw new Error("Could not revoke token")
    }
}
