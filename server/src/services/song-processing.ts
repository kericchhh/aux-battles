import fs from "node:fs/promises";
import path from "node:path";
import {
    CLIP_DURATION_SECONDS,
    FFMPEG_BIN,
    FFPROBE_BIN,
    INCOMING_ROOT,
    SONGS_ROOT,
    WORK_ROOT,
} from "../config/media.js";
import { separateSong } from "../utils/Demucs.js";
import { runProcess } from "../utils/runProcess.js";
import type { ProcessSongJob } from "./song-jobs.js";

const DEMUCS_MODEL = "htdemucs";
const CLIP_BASENAME = "clip";
const STEM_FILES = {
    drumsPath: "drums.mp3",
    bassPath: "bass.mp3",
    melodyPath: "other.mp3",
    vocalsPath: "vocals.mp3",
} as const;

export interface ProcessedSongPaths {
    fullSongPath: string;
    drumsPath: string;
    bassPath: string;
    melodyPath: string;
    vocalsPath: string;
}

function pathInside(root: string, candidate: string) {
    const relative = path.relative(path.resolve(root), path.resolve(candidate));
    return relative !== "" &&
        relative !== ".." &&
        !relative.startsWith(`..${path.sep}`) &&
        !path.isAbsolute(relative);
}

async function validateInputPath(originalPath: string) {
    if (!pathInside(INCOMING_ROOT, originalPath)) {
        throw new Error("Song input path is outside the incoming media directory");
    }

    const input = await fs.realpath(originalPath);
    const incoming = await fs.realpath(INCOMING_ROOT);
    if (!pathInside(incoming, input)) {
        throw new Error("Song input resolves outside the incoming media directory");
    }

    const stats = await fs.stat(input);
    if (!stats.isFile()) throw new Error("Song input is not a regular file");
    return input;
}

export interface Mp3ProbeResult {
    duration: number;
}

export async function probeMp3(
    inputPath: string,
    signal?: AbortSignal,
): Promise<Mp3ProbeResult> {
    const output = await runProcess(FFPROBE_BIN, [
        "-v",
        "error",
        "-show_entries",
        "format=duration,format_name:stream=codec_type,codec_name",
        "-of",
        "json",
        inputPath,
    ], signal);

    let result: {
        format?: { duration?: string; format_name?: string };
        streams?: { codec_type?: string; codec_name?: string }[];
    };
    try {
        result = JSON.parse(output);
    } catch {
        throw new Error("Could not read audio metadata");
    }

    const duration = Number(result.format?.duration);
    const formats = result.format?.format_name?.split(",") ?? [];
    const hasMp3Stream = result.streams?.some(
        (stream) => stream.codec_type === "audio" && stream.codec_name === "mp3",
    );

    if (!Number.isFinite(duration) || duration <= 0 || !formats.includes("mp3") || !hasMp3Stream) {
        throw new Error("The uploaded file is not valid MP3 audio");
    }
    return { duration };
}

export async function probeAudioDuration(
    inputPath: string,
    signal?: AbortSignal,
): Promise<number> {
    return (await probeMp3(inputPath, signal)).duration;
}

export async function extractAudioClip(
    inputPath: string,
    outputPath: string,
    startSeconds: number,
    signal?: AbortSignal,
): Promise<void> {
    await runProcess(FFMPEG_BIN, [
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-ss",
        String(startSeconds),
        "-i",
        inputPath,
        "-t",
        String(CLIP_DURATION_SECONDS),
        "-vn",
        "-ac",
        "2",
        "-ar",
        "44100",
        "-codec:a",
        "libmp3lame",
        "-q:a",
        "2",
        outputPath,
    ], signal);
}

async function requireFile(filePath: string) {
    const stats = await fs.stat(filePath);
    if (!stats.isFile() || stats.size === 0) {
        throw new Error(`Expected output file is empty: ${path.basename(filePath)}`);
    }
}

export async function cleanSongWork(songId: string) {
    await fs.rm(path.join(WORK_ROOT, songId), { recursive: true, force: true });
}

export async function cleanProcessedSong(songId: string) {
    await fs.rm(path.join(SONGS_ROOT, songId), { recursive: true, force: true });
}

export async function removeIncomingSong(originalPath: string) {
    if (pathInside(INCOMING_ROOT, originalPath)) {
        await fs.rm(originalPath, { force: true });
    }
}

export async function processSongAudio(
    job: ProcessSongJob,
    signal?: AbortSignal,
): Promise<ProcessedSongPaths> {
    const inputPath = await validateInputPath(job.originalPath);
    const duration = await probeAudioDuration(inputPath, signal);
    const availableDuration = duration - job.clipStartSeconds;

    if (availableDuration + 0.05 < CLIP_DURATION_SECONDS) {
        throw new Error(
            `The selected clip must contain ${CLIP_DURATION_SECONDS} seconds of audio`,
        );
    }

    const workDirectory = path.join(WORK_ROOT, job.songId);
    const demucsDirectory = path.join(workDirectory, "demucs");
    const resultDirectory = path.join(workDirectory, "result");
    const clipPath = path.join(workDirectory, `${CLIP_BASENAME}.mp3`);
    const finalDirectory = path.join(SONGS_ROOT, job.songId);

    await cleanSongWork(job.songId);
    await fs.mkdir(resultDirectory, { recursive: true });

    try {
        await extractAudioClip(
            inputPath,
            clipPath,
            job.clipStartSeconds,
            signal,
        );
        await requireFile(clipPath);
        await separateSong(clipPath, demucsDirectory, signal);

        const demucsOutput = path.join(
            demucsDirectory,
            DEMUCS_MODEL,
            CLIP_BASENAME,
        );
        await fs.rename(clipPath, path.join(resultDirectory, "full.mp3"));

        for (const outputName of Object.values(STEM_FILES)) {
            const source = path.join(demucsOutput, outputName);
            await requireFile(source);
            await fs.rename(source, path.join(resultDirectory, outputName));
        }

        await fs.mkdir(SONGS_ROOT, { recursive: true });
        await fs.rm(finalDirectory, { recursive: true, force: true });
        await fs.rename(resultDirectory, finalDirectory);
    } finally {
        await cleanSongWork(job.songId);
    }

    return {
        fullSongPath: path.join(finalDirectory, "full.mp3"),
        drumsPath: path.join(finalDirectory, STEM_FILES.drumsPath),
        bassPath: path.join(finalDirectory, STEM_FILES.bassPath),
        melodyPath: path.join(finalDirectory, STEM_FILES.melodyPath),
        vocalsPath: path.join(finalDirectory, STEM_FILES.vocalsPath),
    };
}
