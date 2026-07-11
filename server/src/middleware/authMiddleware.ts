import { getBearer, validateJWT } from "../utils/Auth.js"
import { getUserById } from "../db/queries/users.js"
import { AppError } from "../utils/AppError.js"
import type { Request, Response, NextFunction } from "express"

export async function requireAuth(req: Request, res: Response, next: NextFunction) {
  try {
    const token = getBearer(req)
    const userId = validateJWT(token, process.env.JWT_SECRET!)
    const user = await getUserById(userId)
    if (!user) throw new AppError("User not found", 401)
    req.userId = userId
    next()
  } catch (err) {
    next(err)
  }
}
