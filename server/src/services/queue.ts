import "dotenv/config";
import { PgBoss } from "pg-boss";

export const PROCESS_SONG_QUEUE = "process-song";
export const FAILED_SONG_QUEUE = "process-song-failed";

function requiredEnvironment(name: string): string {
    const value = process.env[name];
    if (!value) {
        throw new Error(`${name} is required`)
    }
    return value;
}

export const queue = new PgBoss({
    connectionString: requiredEnvironment("DATABASE_URL")
});

let startPromise: Promise<PgBoss> | undefined;

async function initialize(): Promise<PgBoss> {
    queue.on("error", (error) => {
        console.error("Job queue error:", error)
    })
    queue.on("warning", (warning) => {
        console.warn("Job queue warning:", warning)
    })
    await queue.start();

    await queue.createQueue(FAILED_SONG_QUEUE);

    await queue.createQueue(PROCESS_SONG_QUEUE, {
        policy: "singleton",
        retryLimit: 2,
        retryDelay: 30,
        retryBackoff: true,
        expireInSeconds: 1000,
        heartbeatSeconds: 60,
        deleteAfterSeconds: 7 * 24 * 60 * 60,
        deadLetter: FAILED_SONG_QUEUE,
        warningQueueSize: 50,
    })
    return queue
}

export function startQueue(): Promise<PgBoss> {
    startPromise ??= initialize();
    return startPromise
}
