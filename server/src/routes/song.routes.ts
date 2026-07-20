import { Router } from "express";
import {
    getAllSongs,
    getSongByID,
    searchSongs,
    addSong,
    patchSong,
    deleteSong
} from "../handlers/song.js";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { requireAdmin } from "../middleware/adminMiddleware.js";
import { uploadSong } from "../middleware/upload.js";

const router = Router();

router.get("/", asyncHandler(getAllSongs));
router.get("/search", asyncHandler(searchSongs));
router.get("/:id", asyncHandler(getSongByID));
router.post("/", requireAdmin,uploadSong.single("song"), asyncHandler(addSong));
router.patch("/:id", requireAdmin, asyncHandler(patchSong));
router.delete("/:id", requireAdmin, asyncHandler(deleteSong));

export default router;
