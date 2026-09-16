import { Router } from "express";
import {
    addSong,
    patchSong,
    deleteSong
} from "../handlers/song.js";
import { getAllSongs, searchSongs, getSongByID, getSongStatus } from "../handlers/song-read.js";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { requireAdmin } from "../middleware/adminMiddleware.js";
import { requireAuth } from "../middleware/authMiddleware.js";
import { uploadSong } from "../middleware/upload.js";
import { rateLimit } from "express-rate-limit";

const router = Router();
const uploadRateLimit = rateLimit({
    windowMs: 60 * 60 * 1000,
    limit: 5,
    standardHeaders: true,
    legacyHeaders: false,
    keyGenerator: (req) => req.userId ?? "anonymous",
    message: { message: "Too many song uploads; try again later" },
});

router.get("/", asyncHandler(getAllSongs));
router.get("/search", asyncHandler(searchSongs));
router.get("/:id/status", requireAuth, asyncHandler(getSongStatus));
router.get("/:id", asyncHandler(getSongByID));
router.post("/", requireAuth, uploadRateLimit, uploadSong.single("song"), asyncHandler(addSong));
router.patch("/:id", requireAdmin, asyncHandler(patchSong));
router.delete("/:id", requireAdmin, asyncHandler(deleteSong));

export default router;
