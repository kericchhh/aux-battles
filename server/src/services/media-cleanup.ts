import fs from "node:fs/promises";
import path from "node:path";
import { INCOMING_ROOT, SONGS_ROOT, WORK_ROOT } from "../config/media.js";
import { getSongById } from "../db/queries/songs.js";

const UUID_PATTERN = /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i;
const ABANDONED_AFTER_MS = 24 * 60 * 60 * 1_000;
const ABANDONED_WORK_AFTER_MS = 6 * 60 * 60 * 1_000;

async function entries(root: string) {
    try {
        return await fs.readdir(root, { withFileTypes: true });
    } catch (error) {
        if ((error as NodeJS.ErrnoException).code === "ENOENT") return [];
        throw error;
    }
}

async function isOlderThan(entryPath: string, ageMs: number) {
    try {
        const stat = await fs.stat(entryPath);
        return Date.now() - stat.mtimeMs > ageMs;
    } catch (error) {
        if ((error as NodeJS.ErrnoException).code === "ENOENT") return false;
        throw error;
    }
}

export async function cleanupAbandonedMedia() {
    let removed = 0;

    for (const entry of await entries(WORK_ROOT)) {
        const entryPath = path.join(WORK_ROOT, entry.name);
        if (await isOlderThan(entryPath, ABANDONED_WORK_AFTER_MS)) {
            await fs.rm(entryPath, { recursive: true, force: true });
            removed += 1;
        }
    }

    for (const entry of await entries(INCOMING_ROOT)) {
        if (!entry.isFile() || !entry.name.startsWith("v2-") || path.extname(entry.name).toLowerCase() !== ".mp3") continue;
        const entryPath = path.join(INCOMING_ROOT, entry.name);
        if (!await isOlderThan(entryPath, ABANDONED_AFTER_MS)) continue;

        const songId = path.parse(entry.name).name.replace(/^v2-/, "");
        const song = UUID_PATTERN.test(songId) ? await getSongById(songId) : undefined;
        if (!song || song.status !== "PROCESSING") {
            await fs.rm(entryPath, { force: true });
            removed += 1;
        }
    }

    for (const entry of await entries(SONGS_ROOT)) {
        if (!entry.isDirectory() || !UUID_PATTERN.test(entry.name)) continue;
        const entryPath = path.join(SONGS_ROOT, entry.name);
        if (!await isOlderThan(entryPath, ABANDONED_AFTER_MS)) continue;

        const song = await getSongById(entry.name);
        if (!song || song.status !== "READY") {
            await fs.rm(entryPath, { recursive: true, force: true });
            removed += 1;
        }
    }

    return removed;
}
