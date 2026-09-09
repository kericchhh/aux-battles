import { Router } from "express";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { requireAuth } from "../middleware/authMiddleware.js";
import { createLobby, getLobby, joinLobby, pickSong } from "../handlers/battles.js";

const router = Router();

router.use(requireAuth)
router.post("/", asyncHandler(createLobby))
router.post("/join", asyncHandler(joinLobby))
router.post("/:battleId/rounds/picks", asyncHandler(pickSong))
router.get("/:battleId", asyncHandler(getLobby))

export default router
