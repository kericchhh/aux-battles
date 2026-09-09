import type { Request, Response } from "express";
import { z } from "zod";
import { getSongs, getSongCatalogById, searchSong } from "../db/queries/songs.js";
import { AppError } from "../utils/AppError.js";

const pagination = z.object({
  limit: z.coerce.number().int().min(1).max(100).default(50),
  offset: z.coerce.number().int().min(0).max(100000).default(0),
});
export async function getAllSongs(req: Request, res: Response) {
  const page = pagination.parse(req.query);
  res.json(await getSongs(undefined, page.limit, page.offset));
}
export async function searchSongs(req: Request, res: Response) {
  const input = pagination.extend({ q: z.string().trim().min(1).max(255) }).parse(req.query);
  res.json(await searchSong(input.q, undefined, input.limit, input.offset));
}
export async function getSongByID(req: Request, res: Response) {
  const id = z.string().uuid().parse(req.params.id);
  const song = await getSongCatalogById(id);
  if (!song) throw new AppError("Song not found", 404);
  res.json(song);
}
