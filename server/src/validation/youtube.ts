import { z } from "zod";
import { songUploadSchema } from "./songs.js";

export const ytDlpInfoSchema = z.object({
    id: z.string().min(1),
    title: z.string().min(1),
    track: z.string().nullish(),
    artist: z.string().nullish(),
    uploader: z.string().nullish(),
    channel: z.string().nullish(),
    duration: z.number().finite().positive(),
    thumbnail: z.string().nullish(),
    is_live: z.boolean().nullish(),
    live_status: z.string().nullish()
})

export const allowedHosts = new Set([
    "youtube.com",
    "www.youtube.com",
    "m.youtube.com",
    "music.youtube.com",
    "youtu.be"
])

function isAllowedYouTubeUrl(value: string) {
    try {
        const url = new URL(value);
        return (
            url.protocol === "https:" && allowedHosts.has(url.hostname.toLowerCase())
        )
    }catch {
        return false
    }
}

export const youtubeUrlSchema = z.string().trim().url().refine(isAllowedYouTubeUrl, {message: "Enter a valid YouTube URL"})

export const youtubeInfoRequestSchema = z.object({
    url: youtubeUrlSchema,
})
export const youtubeImportSchema = songUploadSchema.extend({
    youtubeUrl: youtubeUrlSchema
})
