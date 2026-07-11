import {z} from "zod"
export const submitGuessSchema = z.object({
    guess: z.string().min(1).max(255)
})
