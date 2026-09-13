import type { JobWithMetadata } from "pg-boss";
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

    await patchSongQuery(input.songId, { status: "PROCESSING" });

    try {
        const paths = await processSongAudio(input, job.signal);
        const updated = await patchSongQuery(input.songId, {
            ...paths,
            status: "READY",
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
            await patchSongQuery(input.songId, { status: "FAILED" });
            await cleanProcessedSong(input.songId).catch(() => undefined);
            await removeIncomingSong(input.originalPath).catch(() => undefined);
        }
        throw error;
    }
}

async function runWorker() {
    const boss = await startQueue();
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

    let stopping = false;
    const stop = async (signal: NodeJS.Signals) => {
        if (stopping) return;
        stopping = true;
        console.log(`Received ${signal}; stopping song worker`);
        try {
            await boss.stop({ graceful: true, timeout: 30_000 });
        } catch (error) {
            console.error("Could not stop song worker cleanly:", error);
            process.exitCode = 1;
        }
    };

    process.once("SIGINT", () => void stop("SIGINT"));
    process.once("SIGTERM", () => void stop("SIGTERM"));
}

runWorker().catch((error: unknown) => {
    console.error("Song processing worker failed:", error);
    process.exit(1);
});
