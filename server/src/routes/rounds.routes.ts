import { Router } from "express";
import { requireAuth } from "../middleware/authMiddleware.js";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { submitGuess, getRoundAudio } from "../handlers/rounds.js";

const router = Router()

router.use(requireAuth)
router.get("/:roundId/audio", asyncHandler(getRoundAudio))
router.post("/:roundId/guess", asyncHandler(submitGuess))

export default router
