import { sql } from "drizzle-orm";
import { fromDrizzle } from "pg-boss";
import { z } from "zod";
import type { Transaction } from "../db/types.js";
import {
    PROCESS_SONG_QUEUE,
    queue,
} from "./queue.js";

export const processSongJobSchema = z.object({
    songId: z.string().uuid(),
    originalPath: z.string().min(1),
    clipStartSeconds: z.number().finite().nonnegative(),
});

export type ProcessSongJob = z.infer<typeof processSongJobSchema>;

export async function enqueueSongProcessing(
    job: ProcessSongJob,
    tx: Transaction
) {
   const jobId = await queue.send(
       PROCESS_SONG_QUEUE,
       job,
       {
           id: job.songId,
           db: fromDrizzle(tx, sql)
       }
   );

   if (!jobId){
       throw new Error("Could not enqueue song processing")
   }
   return jobId;
}
