import {z} from "zod";

export const createLobbySchema = z.object({
    rounds: z.number().int().min(1).max(10).optional()
})

export const joinLobbySchema = z.object({
    inviteCode: z.string().length(6)
})
