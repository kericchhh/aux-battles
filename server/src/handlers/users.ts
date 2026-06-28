import type { Response, Request } from "express";
import { userRegisterSchema, userLoginSchema } from "../validation/users.js";
import { getUserByIdentifier, registerUserQuery } from "../db/queries/users.js";
import { hashPassword, makeJWT, makeRefreshToken, validateHash } from "../utils/Auth.js";
import { AppError } from "../utils/AppError.js";
import { saveRefreshToken } from "../db/queries/refresh.js";

export async function registerUser(req: Request, res: Response) {
    const user = userRegisterSchema.parse(req.body)
    const hashed = await hashPassword(user.password)
    const registerUser = await registerUserQuery({
        username: user.username,
        email: user.email,
        avatarUrl: user.avatarUrl,
        role: user.role,
        passwordHash: hashed
    })
    if (!registerUser)throw new AppError("Could not create user", 400)
    res.status(200).json({ id: registerUser.id, email: registerUser.email, role: registerUser.role })
}

export async function loginUser(req: Request, res: Response) {
    const user = userLoginSchema.parse(req.body)
    const loginUser = await getUserByIdentifier(user.identifier)
    if(!loginUser)throw new AppError("Invalid credentials", 404)

    const match = await validateHash(user.password, loginUser.passwordHash)
    if(!match)throw new AppError("Invalid credentials", 404)
    
    const token = makeJWT(loginUser.id, 3600, process.env.JWT_SECRET!)
    const refreshToken = makeRefreshToken()
    const expiresAt = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000)
    const saved = await saveRefreshToken(loginUser.id, refreshToken, expiresAt)
    if(!saved)throw new AppError("Could not save refresh token")
    res.status(200).json({
        id: loginUser.id,
        createdAt: loginUser.createdAt,
        updatedAt: loginUser.updatedAt,
        email: loginUser.email,
        username: loginUser.username,
        token: token,
        refreshToken: refreshToken
    })
}
