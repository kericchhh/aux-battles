import type { Response, Request } from "express";
import { userRegisterSchema, userLoginSchema } from "../validation/users.js";
import { getUserByIdentifier, registerUserQuery } from "../db/queries/users.js";
import { hashPassword, makeJWT, makeRefreshToken, validateHash } from "../utils/Auth.js";
import { AppError } from "../utils/AppError.js";
import { createSession, deleteSession, findSession } from "../services/sessions.js";

export async function registerUser(req: Request, res: Response) {
    const input = userRegisterSchema.parse(req.params);
    const user = await registerUserQuery({
        username: input.username,
        email: input.email,
        avatarUrl: input.avatarUrl ?? null,
        passwordHash: await hashPassword(input.password),
        role: "USER"
    });

    if (!user){
        throw new AppError("Could not create user", 500)
    }

    res.status(201).json({
        id: user.id,
        username: user.username,
        email: user.email
    });
}

export async function loginUser(req: Request, res: Response) {
    const input = userLoginSchema.parse(req.params)
    const user = await getUserByIdentifier(input.identifier);

    if (!user || !(await validateHash(input.password, user.passwordHash))){
        throw new AppError("Invalid credentials", 401)
    }

    await deleteSession(req.headers.cookie, res);
    await createSession(user.id, res);

    res.json({
        id: user.id,
        username: user.username,
        email: user.email
    });
}

export async function getCurrentUser(req: Request, res: Response) {
    res.setHeader("Cache-Control", "no-store");

    const session = await findSession(req.headers.cookie);

    if (!session){
        throw new AppError("Please log in", 401)
    }

    res.json({
        id: session.userId,
        username: session.username,
        email: session.email
    });
}

export async function logoutUser(req: Request, res: Response) {
    await deleteSession(req.headers.cookie, res);
    res.status(204).end();
}
