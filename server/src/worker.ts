import type { JobWithMetadata } from "pg-boss";
import { randomUUID } from "node:crypto";
import { getSongById, patchSongQuery } from "./db/queries/songs.js";
import { PROCESS_SONG_QUEUE, startQueue } from "./services/queue.js";
import {
    processSongJobSchema,
    type ProcessSongJob,
} from "./services/song-jobs.js";
import {
    cleanProcessedSong,
    cleanSongWork,
    processSongAudio,
    removeIncomingSong,
} from "./services/song-processing.js";
import { cleanupAbandonedMedia } from "./services/media-cleanup.js";
import {
    clearWorkerHeartbeat,
    recordWorkerHeartbeat,
    WORKER_HEARTBEAT_INTERVAL_MS,
} from "./services/worker-health.js";

const MEDIA_CLEANUP_INTERVAL_MS = 60 * 60 * 1_000;

function publicProcessingError(error: unknown) {
    const message = error instanceof Error ? error.message : "";
    if (message.includes("selected clip must contain")) return message;
    if (message.includes("valid MP3") || message.includes("audio metadata")) {
        return "The uploaded file is not valid MP3 audio.";
    }
    if (message.includes("demucs")) {
        return "The song could not be separated into audio stems.";
    }
    return "Song processing failed after several attempts.";
}

async function handleSongJob(job: JobWithMetadata<ProcessSongJob>) {
    const input = processSongJobSchema.parse(job.data);
    const song = await getSongById(input.songId);

    if (!song) {
        await cleanSongWork(input.songId);
        await cleanProcessedSong(input.songId);
        await removeIncomingSong(input.originalPath);
        console.warn(`Discarded job ${job.id}: song ${input.songId} no longer exists`);
        return;
    }

    if (
        song.status === "READY" &&
        song.fullSongPath &&
        song.drumsPath &&
        song.bassPath &&
        song.melodyPath &&
        song.vocalsPath
    ) {
        await removeIncomingSong(input.originalPath);
        return;
    }

    await patchSongQuery(input.songId, { status: "PROCESSING", processingError: null });

    try {
        const paths = await processSongAudio(input, job.signal);
        const updated = await patchSongQuery(input.songId, {
            ...paths,
            status: "READY",
            processingError: null,
        });
        if (!updated) throw new Error(`Song ${input.songId} was deleted while processing`);

        await removeIncomingSong(input.originalPath).catch((error: unknown) => {
            console.warn(`Could not remove input for song ${input.songId}:`, error);
        });
        console.log(`Song ${input.songId} is ready`);
    } catch (error) {
        await cleanSongWork(input.songId).catch(() => undefined);
        const finalAttempt = job.retryCount >= job.retryLimit;
        if (finalAttempt) {
            await patchSongQuery(input.songId, {
                status: "FAILED",
                processingError: publicProcessingError(error),
            });
            await cleanProcessedSong(input.songId).catch(() => undefined);
            await removeIncomingSong(input.originalPath).catch(() => undefined);
            console.error(`Song ${input.songId} failed processing:`, error);
        }
        throw error;
    }
}

async function runWorker() {
    const instanceId = randomUUID();
    const boss = await startQueue();
    await recordWorkerHeartbeat(instanceId);

    const removed = await cleanupAbandonedMedia();
    if (removed > 0) console.log(`Removed ${removed} abandoned media entries`);

    await boss.work(
        PROCESS_SONG_QUEUE,
        {
            batchSize: 1,
            includeMetadata: true,
            localConcurrency: 1,
        },
        async (jobs: JobWithMetadata<ProcessSongJob>[]) => {
            const job = jobs[0];
            if (job) await handleSongJob(job);
        },
    );

    console.log("Song processing worker is running");

    const heartbeatTimer = setInterval(() => {
        void recordWorkerHeartbeat(instanceId).catch((error: unknown) => {
            console.error("Could not update song worker heartbeat:", error);
        });
    }, WORKER_HEARTBEAT_INTERVAL_MS);
    const cleanupTimer = setInterval(() => {
        void cleanupAbandonedMedia().then((count) => {
            if (count > 0) console.log(`Removed ${count} abandoned media entries`);
        }).catch((error: unknown) => {
            console.error("Could not clean abandoned media:", error);
        });
    }, MEDIA_CLEANUP_INTERVAL_MS);

    let stopping = false;
    const stop = async (signal: NodeJS.Signals) => {
        if (stopping) return;
        stopping = true;
        clearInterval(heartbeatTimer);
        clearInterval(cleanupTimer);
        console.log(`Received ${signal}; stopping song worker`);
        try {
            await boss.stop({ graceful: true, timeout: 30_000 });
        } catch (error) {
            console.error("Could not stop song worker cleanly:", error);
            process.exitCode = 1;
        } finally {
            await clearWorkerHeartbeat(instanceId).catch((error: unknown) => {
                console.error("Could not clear song worker heartbeat:", error);
            });
        }
    };

    process.once("SIGINT", () => void stop("SIGINT"));
    process.once("SIGTERM", () => void stop("SIGTERM"));
}

runWorker().catch((error: unknown) => {
    console.error("Song processing worker failed:", error);
    process.exit(1);
});
