import { Router } from "express";
import { requireAuth } from "../middleware/authMiddleware.js";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { submitGuess } from "../handlers/rounds.js";

const router = Router()

router.post("/:roundId/guess", requireAuth, asyncHandler(submitGuess))

export default router
