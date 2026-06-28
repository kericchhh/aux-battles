import type { Request, Response } from "express";
import { getSongs, getSongById, searchSong, addSongQuery } from "../db/queries/songs.js";
import { AppError } from "../utils/AppError.js";
import { songCreateSchema, songIdSchema, songSearchSchema } from "../validation/songs.js";

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

export async function searchSongs(req: Request, res: Response) {
    const result = songSearchSchema.safeParse(req.query)
    if(!result.success) throw new AppError("Invalid query", 400);
    const {q} = result.data
    const songs = await searchSong(q)
    if(!songs){
        throw new AppError("No matching songs found", 404)
    } 
    res.json(songs)
}

export async function addSong(req: Request, res: Response) {
    const result = songCreateSchema.safeParse(req.body)
    if(!result.success){
        throw new AppError("Invalid fields", 400)
    }
    const toAdd = await addSongQuery(result.data)
    if(!toAdd){
        throw new AppError("Could not create song, try again", 404)
    }
    res.json(toAdd)
}
