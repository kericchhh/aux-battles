import fs from "node:fs/promises";
import path from "node:path";
import pg from "pg";
import { db } from "../db/index.js";
import { patchSongQuery } from "../db/queries/songs.js";
import { songsTable } from "../db/schema.js";

const apply = process.argv.includes("--apply");
const queuePool = new pg.Pool({ connectionString: process.env.DATABASE_URL });

async function fileAvailable(filePath: string | null) {
    if (!filePath) return false;
    try {
        const stat = await fs.stat(filePath);
        return stat.isFile() && stat.size > 0;
    } catch {
        return false;
    }
}

function usesLegacyDirectory(filePath: string | null) {
    if (!filePath) return false;
    const normalized = path.normalize(filePath);
    return normalized.includes(`${path.sep}uploads${path.sep}`) ||
        normalized.includes(`${path.sep}storage${path.sep}stems${path.sep}`);
}

async function hasActiveProcessingJob(songId: string) {
    try {
        const result = await queuePool.query(
            `select 1
               from pgboss.job
              where name = 'process-song'
                and id = $1::uuid
                and state in ('created', 'retry', 'active')
              limit 1`,
            [songId],
        );
        return result.rowCount === 1;
    } catch (error) {
        if ((error as { code?: string }).code === "42P01") return false;
        throw error;
    }
}

async function main() {
    const songs = await db.select().from(songsTable);
    let stale = 0;

    for (const song of songs) {
        if (song.status === "PROCESSING" && !await hasActiveProcessingJob(song.id)) {
            stale += 1;
            console.log(`${song.id} ${song.artist} - ${song.title}: missing processing job`);
            if (apply) {
                await patchSongQuery(song.id, {
                    status: "FAILED",
                    processingError: "The processing job is no longer available. Upload this song again.",
                });
            }
            continue;
        }
        if (song.status !== "READY") continue;
        const paths = [
            song.fullSongPath,
            song.drumsPath,
            song.bassPath,
            song.melodyPath,
            song.vocalsPath,
        ];
        const legacy = paths.some(usesLegacyDirectory);
        const complete = (await Promise.all(paths.map(fileAvailable))).every(Boolean);
        if (!legacy && complete) continue;

        stale += 1;
        const reason = legacy ? "legacy media paths" : "missing media files";
        console.log(`${song.id} ${song.artist} - ${song.title}: ${reason}`);

        if (apply) {
            await patchSongQuery(song.id, {
                status: "FAILED",
                processingError: "Stored audio is unavailable. Upload this song again.",
                fullSongPath: null,
                drumsPath: null,
                bassPath: null,
                melodyPath: null,
                vocalsPath: null,
            });
        }
    }

    console.log(`${apply ? "Cleaned" : "Found"} ${stale} stale song record${stale === 1 ? "" : "s"}`);
    await queuePool.end();
}

main().then(
    () => process.exit(0),
    (error) => {
        console.error(error);
        process.exit(1);
    },
);
