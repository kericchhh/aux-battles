import type { Request, Response } from "express";
import { getSongs, getSongById, searchSong, addSongQuery, patchSongQuery, deleteSongQuery } from "../db/queries/songs.js";
import { AppError } from "../utils/AppError.js";
import { songCreateSchema, songIdSchema, songPatchSchema, songSearchSchema } from "../validation/songs.js";
import type { songsTable } from "../db/schema.js";
import path from "node:path";
import fs from "node:fs/promises";
import { separateSong } from "../utils/Demucs.js";

export async function getAllSongs(req: Request, res: Response) {
    const songs = await getSongs()
    res.status(200).json(songs)
}

export async function getSongByID(req: Request, res: Response) { 
    const result = songIdSchema.safeParse(req.params)
    if(!result.success) throw new AppError("Invalid id", 400)
    const {id} = result.data
    const song = await getSongById(id)
    if(!song) throw new AppError("Song not found", 404)
    res.json(song)
}

export async function searchSongs(req: Request, res: Response) {
    const result = songSearchSchema.safeParse(req.query)
    if(!result.success) throw new AppError("Invalid query", 400);
    const {q} = result.data
    const songs = await searchSong(q)
    if(!songs) throw new AppError("No matching songs found", 404) 
    res.json(songs)
}

export async function addSong(req: Request, res: Response) {
    if(!req.file){
        throw new AppError("Song file is required", 400)
    }
    const result = songCreateSchema.safeParse(req.body)
    if(!result.success) throw new AppError("Invalid fields", 400)
    const created = await addSongQuery({...result.data, status: "PROCESSING"})
    if(!created) throw new AppError("Could not add song, try again", 404)
    const originalPath = path.resolve(req.file.path)
    res.status(201).json(created)
    processSongStems(created.id, originalPath).catch((err) => {
        console.error(`Stem separation failed for song ${created.id}:`,err)
    })
}

async function processSongStems(songId: string, originalPath: string){ 
    const stemsOutputDir = path.resolve("uploads", "stems", songId)
    await fs.mkdir(stemsOutputDir, {recursive: true})
    try{
        await separateSong(originalPath, stemsOutputDir)
        const filenameNoExt = path.basename(originalPath, path.extname(originalPath))
        const demucsOutDir = path.join(stemsOutputDir, "htdemucs", filenameNoExt)
        await patchSongQuery(songId, {
            status: "READY",
            fullSongPath: originalPath,
            drumsPath: path.join(demucsOutDir,"drums.mp3"),
            bassPath: path.join(demucsOutDir, "bass.mp3"),
            melodyPath: path.join(demucsOutDir, "other.mp3"),
            vocalsPath: path.join(demucsOutDir, "vocals.mp3")
        })
        console.log(`Song ${songId} processed successfully`)
    }catch (err){
        await patchSongQuery(songId, {status: "FAILED"})
        throw err
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
