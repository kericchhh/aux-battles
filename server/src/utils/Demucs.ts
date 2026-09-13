import { DEMUCS_BIN } from "../config/media.js";
import { runProcess } from "./runProcess.js";

export async function separateSong(
    inputPath: string,
    outputDir: string,
    signal?: AbortSignal,
): Promise<void> {
    await runProcess(DEMUCS_BIN, [
        "--name",
        "htdemucs",
        "--mp3",
        "--out",
        outputDir,
        inputPath,
    ], signal);
}
