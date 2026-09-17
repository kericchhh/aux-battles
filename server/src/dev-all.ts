import { spawn, type ChildProcess } from "node:child_process";
import path from "node:path";

const tsxCli = path.resolve("node_modules/tsx/dist/cli.mjs");
const children: ChildProcess[] = [
    spawn(process.execPath, [tsxCli, "watch", "src/server.ts"], { stdio: "inherit" }),
    spawn(process.execPath, [tsxCli, "watch", "src/worker.ts"], { stdio: "inherit" }),
];

let stopping = false;
let closed = 0;

function stop(signal: NodeJS.Signals) {
    if (stopping) return;
    stopping = true;
    for (const child of children) {
        if (child.exitCode === null && child.signalCode === null) child.kill(signal);
    }
}

for (const child of children) {
    child.once("error", (error) => {
        console.error("Could not start development process:", error);
        process.exitCode = 1;
        stop("SIGTERM");
    });
    child.once("exit", (code, signal) => {
        closed += 1;
        if (!stopping) {
            console.error(`Development process stopped (${signal ?? `code ${code ?? "unknown"}`})`);
            process.exitCode = code && code !== 0 ? code : 1;
            stop("SIGTERM");
        }
        if (closed === children.length && process.exitCode === undefined) {
            process.exitCode = 0;
        }
    });
}

process.once("SIGINT", () => stop("SIGINT"));
process.once("SIGTERM", () => stop("SIGTERM"));
