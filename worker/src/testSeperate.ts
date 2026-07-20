import { separateSong } from "./demucs.js";
import path from "node:path";
async function main() {


    await separateSong(
        path.resolve("../server/uploads/originals/Drake - Janice STFU.mp3"),
        path.resolve("../storage/stems")
    );
    console.log("Done!");
}

main().catch(console.error)
