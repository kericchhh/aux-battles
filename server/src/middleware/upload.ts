import multer from "multer";
import fs from "fs";
import { randomUUID } from "crypto";
import { INCOMING_ROOT } from "../config/media.js";

fs.mkdirSync(INCOMING_ROOT, { recursive: true });

const storage = multer.diskStorage({
    destination: (_req, _file, cb) => {
        cb(null, INCOMING_ROOT);
    },

    filename: (_req, _file, cb) => {
        cb(null, `v2-${randomUUID()}.mp3`);
    },
});

export const uploadSong = multer({
    storage,
    limits: {
        files: 1,
        fileSize: 30 * 1024 * 1024,
    },
    fileFilter: (_req, file, cb) => {
        cb(
            null,
            file.mimetype === "audio/mpeg" ||
                file.mimetype === "audio/mp3"
        );
    },
});
