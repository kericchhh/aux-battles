import { eq, ilike } from "drizzle-orm";
import { db } from "../index.js";
import { songsTable } from "../schema.js";

export async function getSongs() {
    const res = await db.select().from(songsTable)
    return res
}

export async function getSongById(id: string) {
    const [res] = await db.select().from(songsTable).where(eq(songsTable.id, id))
    return res
}

export async function searchSong(query: string) {
    const res = await db.select().from(songsTable).where(ilike(songsTable.title, `%${query}%`))
    return res
}

export async function addSongQuery(data: typeof songsTable.$inferInsert) {
    const [res] = await db.insert(songsTable).values(data).returning()
    return res
}
