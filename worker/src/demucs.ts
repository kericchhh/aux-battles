import {spawn} from  "child_process";

export function separateSong(inputPath: string, outputDir: string): Promise<void>{ 
    return new Promise((resolve, reject) => {
        const demucs = spawn("demucs", [
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
