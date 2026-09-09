import type { RequestHandler } from "express";
import { AppError } from "../utils/AppError.js";

export const requireTrustedOrigin: RequestHandler = (req, res, next) => {
  if (["GET", "HEAD", "OPTIONS"].includes(req.method)) {
    next();
    return;
  }

  if (req.get("Origin") !== process.env.CLIENT_ORIGIN) {
    next(new AppError("Untrusted request origin", 403));
    return;
  }

  next();
};
