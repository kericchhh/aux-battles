import {z} from "zod"

export const songIdSchema = z.object({
    id: z.uuid()
})
