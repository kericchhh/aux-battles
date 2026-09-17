import type { ErrorRequestHandler } from "express";
import { ZodError } from "zod";
import multer from "multer";
import { AppError } from "../utils/AppError.js";

function databaseCode(error: unknown, depth = 0): string | undefined {
  if (depth > 4 || !error || typeof error !== "object") return undefined;
  const value = error as { code?: unknown; cause?: unknown };
  return typeof value.code === "string" ? value.code : databaseCode(value.cause, depth + 1);
}
export const errorHandler: ErrorRequestHandler = (error, req, res, next) => {
  if (res.headersSent) { next(error); return; }
  if (error instanceof ZodError) {
    res.status(400).json({ message: "Invalid input", issues: error.issues.map(({ path, message }) => ({ path, message })) });
    return;
  }
  if (error instanceof multer.MulterError) {
    const tooLarge = error.code === "LIMIT_FILE_SIZE";
    res.status(tooLarge ? 413 : 400).json({
      message: tooLarge ? "MP3 files must be 30 MB or smaller" : "Invalid song upload",
    });
    return;
  }
  if (error instanceof AppError) { res.status(error.statusCode).json({ message: error.message }); return; }
  const code = databaseCode(error);
  if (code === "23505" || code === "23503" || code === "23514") {
    res.status(409).json({ message: "The action conflicts with existing data or game state" }); return;
  }
  console.error(error);
  res.status(500).json({ message: "Internal server error" });
};
