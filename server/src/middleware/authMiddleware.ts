import type { RequestHandler } from "express";

import { findSession } from "../services/sessions.js";
import { AppError } from "../utils/AppError.js";

export const requireAuth: RequestHandler = async (req,res,next) => {
    try{
        const session = await findSession(req.headers.cookie);

        if (!session){
            throw new AppError("Please sign in", 401)
        }
        req.userId = session.userId;
        next();
    } catch (error) {
        next(error)
    }
}
