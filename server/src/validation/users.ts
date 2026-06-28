import { z } from "zod";

export const userRegisterSchema = z.object({
    username: z.string().max(30),
    email: z.string().max(255),
    password: z.string()
        .min(8)
        .max(255)
        .regex(/[A-Z]/, "Must contain at least one uppercase letter")
        .regex(/[a-z]/, "Must contain at least one lowercase letter")
        .regex(/[0-9]/, "Must contain at least one number")
        .regex(/[!@#$%^&*]/, "Must contain at least one special character"),
    avatarUrl: z.string().optional(),
    role: z.enum(["USER", "ADMIN"]).default("USER")
})

export const userLoginSchema = z.object({
    identifier: z.string(),
    password: z.string()
})
