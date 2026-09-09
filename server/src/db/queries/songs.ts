import { eq, ilike, and } from "drizzle-orm";
import { db } from "../index.js";
import { songsTable } from "../schema.js";
import  type { Transaction, DbExecutor } from "../types.js"

const catalog = {
  id: songsTable.id, title: songsTable.title, artist: songsTable.artist,
  genre: songsTable.genre, coverImage: songsTable.coverImage,
  album: songsTable.album, duration: songsTable.duration, status: songsTable.status,
};
function pageSize(limit: number) {
  return Number.isFinite(limit) ? Math.max(1, Math.min(100, Math.trunc(limit))) : 50;
}
function pageOffset(offset: number) {
  return Number.isFinite(offset) ? Math.max(0, Math.trunc(offset)) : 0;
}

export async function getSongs(executor: DbExecutor = db, limit = 50, offset = 0) {
   return await executor
        .select()
        .from(songsTable)
        .where(eq(songsTable.status,"READY"))
        .orderBy(songsTable.title, songsTable.id).limit(pageSize(limit)).offset(pageOffset(offset))
}

export async function getSongById(id: string, executor: DbExecutor = db) {
    const [res] = await executor.select().from(songsTable).where(eq(songsTable.id, id))
    return res
}

export async function getSongCatalogById(id: string, executor: DbExecutor = db) {
    const [res] = await executor
        .select(catalog).from(songsTable)
        .where(and(
            eq(songsTable.id, id),
            eq(songsTable.status, "READY")
        ))
    return res
}

export async function searchSong(query: string, executor: DbExecutor = db, limit = 50, offset = 0) {
    const term = query.trim();
    if (!term) return [];
    const escaped = term.replace(/[\\%_]/g, "\\$&");
    return executor.select(catalog).from(songsTable)
        .where(and(
            eq(songsTable.status, "READY"),
            ilike(songsTable.title, `%${escaped}%`)
        ))
        .orderBy(songsTable.title, songsTable.id).limit(pageSize(limit)).offset(pageOffset(offset))
}

export async function addSongQuery(data: typeof songsTable.$inferInsert, executor: DbExecutor = db) {
    const [res] = await executor.insert(songsTable).values(data).returning()
    return res
}

export async function patchSongQuery(
    id: string, 
    data: Partial<Omit<typeof songsTable.$inferInsert, "id">>,
    executor: DbExecutor = db,
) {
   const [res] = await executor.update(songsTable).set(data).where(eq(songsTable.id, id)).returning()
   return res
}

export async function deleteSongQuery(id: string, executor: DbExecutor = db) {
    const [res] = await executor.delete(songsTable).where(eq(songsTable.id, id)).returning()
    return res
}

export async function getSongForShare(songId: string, tx: Transaction) {
    const [res] = await tx.select().from(songsTable).where(eq(songsTable.id, songId)).for("share")
    return res
}
