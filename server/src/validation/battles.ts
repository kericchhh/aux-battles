import {z} from "zod";

export const createLobbySchema = z.object({
    rounds: z.number().int().min(1).max(10).optional()
})
