import type { Request, Response } from "express";
import { addSongQuery, patchSongQuery, deleteSongQuery } from "../db/queries/songs.js";
import { AppError } from "../utils/AppError.js";
import { songIdSchema, songPatchSchema, songUploadSchema } from "../validation/songs.js";
import type { songsTable } from "../db/schema.js";
import path from "node:path";
import fs from "node:fs/promises";
import { db } from "../db/index.js";
import { enqueueSongProcessing } from "../services/song-jobs.js";

export async function addSong(req: Request, res: Response) {
    if(!req.file){
        throw new AppError("Song file is required", 400)
    }
    const originalPath = path.resolve(req.file.path)
    const result = songUploadSchema.safeParse(req.body)
    if(!result.success){
        await fs.rm(originalPath, {force: true});
        throw new AppError("Invalid fields", 400)
    }
    const { clipStartSeconds, ...metadata } = result.data
    try{
        const created = await db.transaction(async (tx) => {
            const song = await addSongQuery({ ...metadata, status: "PROCESSING"},tx)
            if (!song) {
                throw new AppError("Could not create song", 500)
            }
            await enqueueSongProcessing({
                songId: song.id,
                originalPath,
                clipStartSeconds,
            }, tx)
            return song
        })
        res.status(202).json({
            id: created.id,
            status: created.status
        })
    }catch (error) {
        await fs.rm(originalPath, {force: true})
        throw error
    }
}

export async function patchSong(req: Request, res: Response) {
    const result = songPatchSchema.safeParse(req.body)
    const resultId = songIdSchema.safeParse(req.params)
    if(!resultId.success) throw new AppError("Invalid song id", 400)
    if(!result.success) throw new AppError("Invalid field/s", 400)
    const { id } = resultId.data
    const toPatch = await patchSongQuery(id,result.data as Partial<typeof songsTable.$inferInsert>) 
    if(!toPatch) throw new AppError("Could not update song", 404)
    res.json(toPatch)
}

export async function deleteSong(req: Request, res: Response) {
    const result = songIdSchema.safeParse(req.params)
    if(!result.success) throw new AppError("Invalid song id", 400)
    const { id } = result.data
    const toDelete = await deleteSongQuery(id)
    if(!toDelete) throw new AppError("Could not delete song", 404)
    res.json(toDelete)
}
