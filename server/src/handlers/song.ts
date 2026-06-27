import type { Request, Response } from "express";
import { getSongs, getSongById } from "../db/queries/songs.js";
import { AppError } from "../utils/AppError.js";
import { songIdSchema } from "../validation/songs.js";

export async function getAllSongs(req: Request, res: Response) {
    const songs = await getSongs()
    res.status(200).json(songs)
}

export async function getSongByID(req: Request, res: Response) { 
    const result = songIdSchema.safeParse(req.params)
    if(!result.success){
        throw new AppError("Invalid id", 400)
    }
    const {id} = result.data
    const song = await getSongById(id)
    if(!song){
        throw new AppError("Song not found", 404)
    }
    res.json(song)
}
