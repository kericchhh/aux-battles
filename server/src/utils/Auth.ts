import * as argon2 from "argon2";
import jwt from "jsonwebtoken";
import type { JwtPayload } from "jsonwebtoken";
import type { Request, Response } from "express";
import crypto from "crypto"
import { AppError } from "./AppError.js";
import { revokeRefreshToken, userForRefreshToken } from "../db/queries/refresh.js";

export async function hashPassword(pswd: string): Promise<string> {
   try{
       const hash = await argon2.hash(pswd)
       return hash
   }catch (err){
       console.error(err)
       throw err
   } 
}
export async function validateHash(pswd: string, hash: string): Promise<boolean> {
    try{
        if(await argon2.verify(hash,pswd)) return true
        else return false
    }catch (err){
        console.error(err)
        throw err
    }
}

type Payload = Pick<JwtPayload, "sub" | "iat" | "exp">

export function makeJWT(userId: string, expiresIn: number, secret: string): string {
   const issuedAt = Math.floor(Date.now() / 1000) 
   const payload: Payload = {
       sub: userId,
       iat: issuedAt,
       exp: issuedAt + expiresIn
   }
   const res = jwt.sign(payload, secret, {algorithm: "HS256"})
   return res
}

export function validateJWT(token: string, secret: string) {
    let decoded: Payload;
    try{
        decoded = jwt.verify(token, secret) as JwtPayload
    }catch (e){
        throw new AppError("Invalid token", 401)
    }

    if(!decoded.sub) throw new AppError("No userId in token", 401)
    return decoded.sub
}

export function getBearer(req: Request): string {
    const authHeader = req.get("Authorization")
    if(!authHeader || !authHeader.startsWith("Bearer ")){
        throw new AppError("No token found", 401)
    }
    return authHeader.slice(7)
}

export function makeRefreshToken() {
    return crypto.randomBytes(32).toString("hex")
}

export async function handlerRefresh(req: Request, res: Response) {
    let refreshToken = getBearer(req)
    const result = await userForRefreshToken(refreshToken)
    if(!result) throw new AppError("Invalid refresh token", 401)
    const user = result.user
    const accessToken = makeJWT(user.id,3600,process.env.JWT_SECRET!)    
    res.status(200).json({ token: accessToken})
}

export async function handlerRevoke(req: Request, res: Response) {
    const refreshToken = getBearer(req)
    await revokeRefreshToken(refreshToken)
    res.status(204).send()
}
