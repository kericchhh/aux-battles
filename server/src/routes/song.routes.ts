import { Router } from "express";
import {
    getAllSongs,
    getSongByID,
    searchSongs,
    addSong,
    // updateSong,
    // deleteSong,
} from "../handlers/song.js";
import { asyncHandler } from "../utils/AsyncHandler.js";

const router = Router();

router.get("/", asyncHandler(getAllSongs));
router.get("/search", asyncHandler(searchSongs));
router.get("/:id", asyncHandler(getSongByID));
router.post("/", asyncHandler(addSong));
// router.patch("/:id", updateSong);
// router.delete("/:id", deleteSong);

export default router;
