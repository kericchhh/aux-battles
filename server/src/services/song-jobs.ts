import { sql } from "drizzle-orm";
import { fromDrizzle } from "pg-boss";
import type { Transaction } from "../db/types.js";
import {
    PROCESS_SONG_QUEUE,
    queue,
} from "./queue.js";

export interface ProcessSongJob {
    songId: string;
    originalPath: string;
    clipStartSeconds: number;
}

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
