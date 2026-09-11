import { z } from "zod";

export const userRegisterSchema = z.object({
    username: z.string().trim().min(3).max(30).regex(/^[a-zA-Z0-9_]+$/).transform((value) => value.toLowerCase()),
    email: z.string().trim().email().max(255).transform((value) => value.toLowerCase()),
    password: z.string()
        .min(8)
        .max(255)
        .regex(/[A-Z]/, "Must contain at least one uppercase letter")
        .regex(/[a-z]/, "Must contain at least one lowercase letter")
        .regex(/[0-9]/, "Must contain at least one number")
        .regex(/[!@#$%^&*]/, "Must contain at least one special character"),
    avatarUrl: z.string().optional()
})

export const userLoginSchema = z.object({
    identifier: z.string().trim().min(1).max(255).transform((value) => value.toLowerCase()),
    password: z.string()
})

export const userIdSchema = z.object({
    id: z.string().uuid()
})
