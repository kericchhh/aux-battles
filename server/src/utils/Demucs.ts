import {spawn} from  "child_process";
import path from "node:path"
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const DEMUCS_BIN = path.resolve(__dirname, "..", "..", "..", "worker", "venv", "bin", "demucs")

export function separateSong(inputPath: string, outputDir: string): Promise<void>{ 
    return new Promise((resolve, reject) => {
        const demucs = spawn(DEMUCS_BIN, [
            "--mp3",
            inputPath,
            "-o",
            outputDir
        ]);

        demucs.stdout.on("data", (data) => {
            console.log(data.toString())
        });

        demucs.stderr.on("data", (data) => {
            console.error(data.toString())
        });

        demucs.on("close", (code) => {
            if(code === 0){
                resolve()
            }else{
                reject(new Error(`Demucs exited with code ${code}`))
            }
        })

        demucs.on("error",reject)
    })
}
