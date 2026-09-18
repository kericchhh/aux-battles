import type { Request, Response } from "express";
import { addSongQuery, patchSongQuery, deleteSongQuery } from "../db/queries/songs.js";
import { AppError } from "../utils/AppError.js";
import { songIdSchema, songPatchSchema, songUploadSchema } from "../validation/songs.js";
import type { songsTable } from "../db/schema.js";
import path from "node:path";
import fs from "node:fs/promises";
import { db } from "../db/index.js";
import { enqueueSongProcessing } from "../services/song-jobs.js";
import { CLIP_DURATION_SECONDS, INCOMING_ROOT } from "../config/media.js";
import { probeMp3, cleanProcessedSong, removeIncomingSong } from "../services/song-processing.js";
import { randomUUID } from "node:crypto";
import { youtubeImportSchema, youtubeInfoRequestSchema } from "../validation/youtube.js";
import { downloadYouTubeClip, getYouTubeInfo } from "../services/youtube.js";

interface QueueSongInput {
    songId: string;
    title: string;
    artist: string;
    genre: string;
    album?: string | undefined;
    coverImage?: string | undefined;
    duration: number;
    originalPath: string;
    clipStartSeconds: number
}

async function createQueuedSong(input: QueueSongInput) {
    return db.transaction(async (tx) => {
        const song = await addSongQuery(
            {
                id: input.songId,
                title: input.title,
                artist: input.artist,
                genre: input.genre,
                album: input.album,
                coverImage: input.coverImage,
                duration: Math.max(1, Math.round(input.duration)),
                status: "PROCESSING",
                processingError: null
            }, tx
        )
        if (!song) throw new AppError("Could not create song", 500);
        await enqueueSongProcessing(
            {
                songId: song.id,
                originalPath: input.originalPath,
                clipStartSeconds: input.clipStartSeconds
            }, tx
        )
        return song
    })
}

export async function addSong(req: Request, res: Response) {
    if (!req.file) {
        throw new AppError("Song file is required", 400)
    }
    const originalPath = path.resolve(req.file.path)
    const songId = path.parse(req.file.filename).name.replace(/^v2-/, "");
    const result = songUploadSchema.safeParse(req.body)
    if (!result.success) {
        await fs.rm(originalPath, { force: true });
        throw new AppError("Invalid fields", 400)
    }
    const { clipStartSeconds, ...metadata } = result.data
    let duration: number;
    try {
        duration = (await probeMp3(originalPath)).duration;
    } catch {
        await fs.rm(originalPath, { force: true });
        throw new AppError("Upload a valid MP3 file", 400);
    }
    if (duration - clipStartSeconds + 0.05 < CLIP_DURATION_SECONDS) {
        await fs.rm(originalPath, { force: true });
        throw new AppError(`The selected clip must contain ${CLIP_DURATION_SECONDS} seconds of audio`, 400);
    }
    try {
        const created = await createQueuedSong({
            songId,
            ...metadata,
            duration,
            originalPath,
            clipStartSeconds
        });

        res.status(202).json({ id: created.id, status: created.status })
    } catch (error) {
        await fs.rm(originalPath, { force: true })
        throw error
    }
}

export async function patchSong(req: Request, res: Response) {
    const result = songPatchSchema.safeParse(req.body)
    const resultId = songIdSchema.safeParse(req.params)
    if (!resultId.success) throw new AppError("Invalid song id", 400)
    if (!result.success) throw new AppError("Invalid field/s", 400)
    const { id } = resultId.data
    const toPatch = await patchSongQuery(id, result.data as Partial<typeof songsTable.$inferInsert>)
    if (!toPatch) throw new AppError("Could not update song", 404)
    res.json(toPatch)
}

export async function deleteSong(req: Request, res: Response) {
    const result = songIdSchema.safeParse(req.params)
    if (!result.success) throw new AppError("Invalid song id", 400)
    const { id } = result.data
    const toDelete = await deleteSongQuery(id)
    if (!toDelete) throw new AppError("Could not delete song", 404)
    await Promise.all([
        cleanProcessedSong(id),
        removeIncomingSong(path.join(INCOMING_ROOT, `v2-${id}.mp3`)),
    ]).catch((error: unknown) => {
        console.warn(`Song ${id} was deleted, but its media cleanup failed:`, error);
    });
    res.json(toDelete)
}

export async function readYouTubeInfo(
    req: Request,
    res: Response,
) {
    const result = youtubeInfoRequestSchema.safeParse(req.body);

    if (!result.success) {
        throw new AppError("Enter a valid YouTube URL", 400);
    }

    try {
        const info = await getYouTubeInfo(result.data.url);
        res.json(info);
    } catch (error) {
        const message =
            error instanceof Error
                ? error.message
                : "Could not read the YouTube video";

        throw new AppError(message, 400);
    }
}

export async function addSongFromYouTube(
    req: Request,
    res: Response,
) {
    const result = youtubeImportSchema.safeParse(req.body);

    if (!result.success) {
        throw new AppError("Invalid YouTube song fields", 400);
    }

    const {
        youtubeUrl,
        clipStartSeconds,
        ...metadata
    } = result.data;

    let info;

    try {
        info = await getYouTubeInfo(youtubeUrl);
    } catch (error) {
        const message =
            error instanceof Error
                ? error.message
                : "Could not read the YouTube video";

        throw new AppError(message, 400);
    }

    if (
        info.duration - clipStartSeconds + 0.05 <
        CLIP_DURATION_SECONDS
    ) {
        throw new AppError(
            `The selected clip must contain ${CLIP_DURATION_SECONDS} seconds of audio`,
            400,
        );
    }

    const songId = randomUUID();
    const outputPath = path.join(
        INCOMING_ROOT,
        `v2-${songId}.mp3`,
    );

    try {
        const downloaded = await downloadYouTubeClip(
            youtubeUrl,
            outputPath,
            clipStartSeconds,
        );

        await probeMp3(downloaded.originalPath);

        const created = await createQueuedSong({
            songId,
            ...metadata,
            duration: info.duration,
            originalPath: downloaded.originalPath,
            clipStartSeconds:
                downloaded.localClipStartSeconds,
        });

        res.status(202).json({
            id: created.id,
            status: created.status,
        });
    } catch (error) {
        await fs.rm(outputPath, { force: true });

        if (error instanceof AppError) {
            throw error;
        }

        const message =
            error instanceof Error
                ? error.message
                : "Could not import the YouTube video";

        throw new AppError(message, 400);
    }
}
