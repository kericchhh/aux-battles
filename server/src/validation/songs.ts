import {z} from "zod"

export const songIdSchema = z.object({
    id: z.uuid()
})

export const songSearchSchema = z.object({
    q: z.string().min(1)
})

export const songCreateSchema = z.object({
    title: z.string().trim().min(1).max(255),
    artist: z.string().trim().min(1).max(255),
    genre: z.string().trim().min(1).max(50),
    coverImage: z.string().optional(),
    album: z.string().max(255).optional(),
    duration: z.coerce.number().int().positive(),
    status: z.enum(["PROCESSING", "READY", "FAILED"]).default("PROCESSING")
})

export const songPatchSchema = songCreateSchema.partial()

export const songUploadSchema = songCreateSchema.omit({status: true}).extend({clipStartSeconds: z.coerce.number().min(0)});
