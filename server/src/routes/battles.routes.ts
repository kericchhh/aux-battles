import { Router } from "express";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { requireAuth } from "../middleware/authMiddleware.js";
import { createLobby } from "../handlers/battles.js";

const router = Router();

router.post("/", requireAuth, asyncHandler(createLobby))


export default router
