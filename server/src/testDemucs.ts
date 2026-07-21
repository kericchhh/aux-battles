import { separateSong } from "./utils/Demucs.js";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const inputPath = path.resolve(__dirname, "..", "uploads", "originals", "Drake - Janice STFU.mp3");
const outputDir = path.resolve(__dirname, "..", "uploads", "stems", "test-output");

console.log("Input:", inputPath);
console.log("Output dir:", outputDir);

separateSong(inputPath, outputDir)
    .then(() => console.log("Done! Check the output folder."))
    .catch((err) => console.error("Failed:", err));
