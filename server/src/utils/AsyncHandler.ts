import type { Request, Response, NextFunction } from "express";

type AsyncRoute = (req: Request, res: Response, next: NextFunction) => Promise<void>

export function asyncHandler(handler: AsyncRoute) {
    return (req: Request, res: Response, next: NextFunction) => {
        Promise.resolve().then(() => handler(req,res,next)).catch(next);
    }
}
