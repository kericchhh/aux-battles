import {z} from "zod"

export const songIdSchema = z.object({
    id: z.uuid()
})

export const songSearchSchema = z.object({
    q: z.string().min(1)
})

export const songCreateSchema = z.object({
    title: z.string().max(255),
    artist: z.string().max(255),
    genre: z.string().max(50),
    coverImage: z.string().optional(),
    album: z.string().max(255).optional(),
    duration: z.number(),
    status: z.enum(["PROCESSING", "READY", "FAILED"]).default("PROCESSING")
})
