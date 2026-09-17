import { and, eq } from "drizzle-orm";
import { db } from "../db/index.js";
import { workerHeartbeatsTable } from "../db/schema.js";

export const SONG_WORKER_NAME = "song-processing";
export const WORKER_HEARTBEAT_INTERVAL_MS = 5_000;
const WORKER_STALE_AFTER_MS = 15_000;

export async function recordWorkerHeartbeat(instanceId: string) {
    const now = new Date();
    await db
        .insert(workerHeartbeatsTable)
        .values({ name: SONG_WORKER_NAME, instanceId, lastSeenAt: now })
        .onConflictDoUpdate({
            target: workerHeartbeatsTable.name,
            set: { instanceId, lastSeenAt: now },
        });
}

export async function clearWorkerHeartbeat(instanceId: string) {
    await db
        .delete(workerHeartbeatsTable)
        .where(and(
            eq(workerHeartbeatsTable.name, SONG_WORKER_NAME),
            eq(workerHeartbeatsTable.instanceId, instanceId),
        ));
}

export async function getSongWorkerHealth() {
    const [heartbeat] = await db
        .select({ lastSeenAt: workerHeartbeatsTable.lastSeenAt })
        .from(workerHeartbeatsTable)
        .where(eq(workerHeartbeatsTable.name, SONG_WORKER_NAME));

    const lastSeenAt = heartbeat?.lastSeenAt ?? null;
    return {
        available: Boolean(
            lastSeenAt && Date.now() - lastSeenAt.getTime() <= WORKER_STALE_AFTER_MS
        ),
        lastSeenAt,
    };
}
