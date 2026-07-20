import multer from "multer";
import path from "path";
import fs from "fs";

const uploadDir = "../../uploads/originals/";

fs.mkdirSync(uploadDir, { recursive: true });

const storage = multer.diskStorage({
    destination: (_req, _file, cb) => {
        cb(null, uploadDir);
    },

    filename: (_req, file, cb) => {
        const ext = path.extname(file.originalname);
        cb(null, `${Date.now()}${ext}`);
    },
});

export const uploadSong = multer({
    storage,
    fileFilter: (_req, file, cb) => {
        if (
            file.mimetype === "audio/mpeg" ||
            file.mimetype === "audio/mp3"
        ) {
            cb(null, true);
        } else {
            cb(new Error("Only MP3 files are allowed"));
        }
    },
});
