import { Router } from "express";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { requireAuth } from "../middleware/authMiddleware.js";
import { createLobby, joinLobby, pickSong } from "../handlers/battles.js";

const router = Router();

router.post("/", requireAuth, asyncHandler(createLobby))
router.post("/join", requireAuth, asyncHandler(joinLobby))
router.post("/:battleId/rounds/picks", requireAuth, asyncHandler(pickSong))

export default router
