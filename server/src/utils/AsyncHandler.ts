import type { Request, Response, NextFunction } from "express";

export function asyncHandler(handler: (req: Request, res: Response) => Promise<void>){
    return (req:Request, res: Response, next: NextFunction) => {
        Promise.resolve(handler(req,res)).catch(next)
    }
}
