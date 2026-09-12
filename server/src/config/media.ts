import path from "node:path";
import { fileURLToPath } from "node:url";

const currentDirectory = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(currentDirectory, "../../..");

export const MEDIA_ROOT = path.resolve(
    process.env.MEDIA_ROOT ?? path.join(repositoryRoot, "storage")
);

export const INCOMING_ROOT = path.join(MEDIA_ROOT, "incoming");
export const SONGS_ROOT = path.join(MEDIA_ROOT, "songs");
export const WORK_ROOT = path.join(MEDIA_ROOT, "work");

export const FFMPEG_BIN = process.env.FFMPEG_BIN ?? "ffmpeg";
export const FFPROBE_BIN = process.env.FFPROBE_BIN ?? "ffprobe";

export const DEMUCS_BIN = path.resolve(
    process.env.DEMUCS_BIN ??
        path.join(repositoryRoot, "worker", "venv", "bin", "demucs")
);

export const CLIP_DURATION_SECONDS = 12;
