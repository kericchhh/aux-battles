import { createHash, randomBytes } from "node:crypto";
import { parseCookie } from "cookie";
import {and, eq, gt} from "drizzle-orm";
import type { Response } from "express";

import { db } from "../db/index.js";
import { sessionsTable, usersTable } from "../db/schema.js";

const production = process.env.NODE_ENV === "production";

export const SESSION_COOKIE = production ? "__Host-aux_session" : "aux_session";

const SESSION_DURATION_MS = 7 * 24 * 60 * 60 * 1000;

const cookieOptions = {
    httpOnly: true,
    secure: production,
    sameSite: "lax" as const,
    path: "/"
};

function hashToken(token: string) {
    return createHash("sha256").update(token).digest("hex")
};

export function readSessionToken(cookieHeader?: string) {
    const token = parseCookie(cookieHeader ?? "")[SESSION_COOKIE];

    return token && /^[a-f0-9]{64}$/.test(token) ? token : null;
}
export async function findSession(cookieHeader?: string) {
    const token = readSessionToken(cookieHeader)
    if (!token) return null;

    const [session] = await db
        .select({
            tokenHash: sessionsTable.tokenHash,
            expiresAt: sessionsTable.expiresAt,
            userId: usersTable.id,
            username: usersTable.username,
            email: usersTable.email,
            role: usersTable.role
        })
        .from(sessionsTable)
        .innerJoin(usersTable, eq(usersTable.id, sessionsTable.userId))
        .where(and(eq(sessionsTable.tokenHash,hashToken(token)), gt(sessionsTable.expiresAt, new Date())))

        return session ?? null
}

export async function createSession(userId: string, res: Response) {
    const token = randomBytes(32).toString("hex");
    const expiresAt = new Date(Date.now() + SESSION_DURATION_MS);

    await db.insert(sessionsTable).values({
        tokenHash: hashToken(token),
        userId,
        expiresAt
    })

    res.cookie(SESSION_COOKIE, token, {
        ...cookieOptions,
        maxAge: SESSION_DURATION_MS,
    });
}

export async function deleteSession(cookieHeader: string | undefined, res: Response) {
    const token = readSessionToken(cookieHeader);
    const tokenHash = token ? hashToken(token) : null;

    if (tokenHash) {
        await db.delete(sessionsTable).where(eq(sessionsTable.tokenHash, tokenHash))
    }
    res.clearCookie(SESSION_COOKIE, cookieOptions);
    return tokenHash;
}
