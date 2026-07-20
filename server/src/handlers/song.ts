import type { Request, Response } from "express";
import { getSongs, getSongById, searchSong, addSongQuery, patchSongQuery, deleteSongQuery } from "../db/queries/songs.js";
import { AppError } from "../utils/AppError.js";
import { songCreateSchema, songIdSchema, songPatchSchema, songSearchSchema } from "../validation/songs.js";
import type { songsTable } from "../db/schema.js";

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

    const toAdd = await addSongQuery(result.data)
    if(!toAdd) throw new AppError("Could not create song, try again", 404)
    res.json(toAdd)
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
