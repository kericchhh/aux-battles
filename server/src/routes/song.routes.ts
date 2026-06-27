import { Router } from "express";
import {
    getAllSongs,
    getSongByID,
    // searchSongs,
    // createSong,
    // updateSong,
    // deleteSong,
} from "../handlers/song.js";
import { asyncHandler } from "../utils/AsyncHandler.js";

const router = Router();

router.get("/", asyncHandler(getAllSongs));
router.get("/:id", asyncHandler(getSongByID));
// router.get("/search", searchSongs);
// router.post("/", createSong);
// router.patch("/:id", updateSong);
// router.delete("/:id", deleteSong);

export default router;
