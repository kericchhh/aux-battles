import { Router } from "express";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { requireAuth } from "../middleware/authMiddleware.js";
import { createLobby, getLobby, joinLobby, submitLineup } from "../handlers/battles.js";

const router = Router();

router.use(requireAuth)
router.post("/", asyncHandler(createLobby))
router.post("/join", asyncHandler(joinLobby))
router.put("/:battleId/lineup", asyncHandler(submitLineup))
router.get("/:battleId", asyncHandler(getLobby))

export default router
