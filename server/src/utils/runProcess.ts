import { spawn } from "node:child_process";

const OUTPUT_LIMIT = 32 * 1024;

function appendOutput(current: string, chunk: Buffer | string) {
    const combined = current + chunk.toString();
    return combined.length <= OUTPUT_LIMIT
        ? combined
        : combined.slice(combined.length - OUTPUT_LIMIT);
}

export async function runProcess(
    command: string,
    args: string[],
    signal?: AbortSignal,
): Promise<string> {
    return new Promise((resolve, reject) => {
        let stdout = "";
        let stderr = "";
        let settled = false;

        const child = spawn(command, args, {
            signal,
            windowsHide: true,
            stdio: ["ignore", "pipe", "pipe"],
        });

        child.stdout.on("data", (chunk: Buffer) => {
            stdout = appendOutput(stdout, chunk);
        });
        child.stderr.on("data", (chunk: Buffer) => {
            stderr = appendOutput(stderr, chunk);
        });

        child.once("error", (error) => {
            if (settled) return;
            settled = true;
            reject(error);
        });

        child.once("close", (code, terminationSignal) => {
            if (settled) return;
            settled = true;

            if (code === 0) {
                resolve(stdout);
                return;
            }

            const reason = terminationSignal
                ? `signal ${terminationSignal}`
                : `code ${code ?? "unknown"}`;
            const details = stderr.trim();
            reject(new Error(
                `${command} exited with ${reason}${details ? `: ${details}` : ""}`,
            ));
        });
    });
}
