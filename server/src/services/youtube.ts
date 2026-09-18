import fs from "node:fs/promises";
import path from "node:path";
import { CLIP_DURATION_SECONDS, YT_DLP_BIN } from "../config/media.js";
import { runProcess } from "../utils/runProcess.js";
import { ytDlpInfoSchema } from "../validation/youtube.js";

const MAX_VIDEO_DURATION_SECONDS = 20 * 60;
const MAX_DOWNLOAD_SIZE_BYTES = 30 * 1024 * 1024;

export interface YouTubeInfo {
    videoId: string;
    title: string;
    artist: string;
    duration: number;
    thumbnail?: string;
}

export interface DownloadedYouTubeClip {
    originalPath: string;
    localClipStartSeconds: number;
}

export async function getYouTubeInfo(url: string, signal?: AbortSignal): Promise<YouTubeInfo> {
    let output: string;

    try{
        output = await runProcess(
            YT_DLP_BIN,
            [
                "--ignore-config",
                "--no-playlist",
                "--dump-single-json",
                "--skip-download",
                "--",
                url
            ],
            signal
        )
    }catch (error){
        throw new Error("Could not read the YouTube video", {
            cause: error
        })
    };

    let json: unknown;

    try{
        json = JSON.parse(output)
    }catch (error){
        throw new Error("yt-dlp returned invalid metadata", {
            cause: error
        })
    };

    const result = ytDlpInfoSchema.safeParse(json);
    if (!result.success) throw new Error("The YouTube video metadata is incomplete");
    const video = result.data;
    if(
        video.is_live === true ||
        video.live_status === "is_live" ||
        video.live_status === "is_upcoming"
    ){
        throw new Error("Live and upcoming videos are not supported")
    };
    if(video.duration < CLIP_DURATION_SECONDS) throw new Error(`The video must be at least ${CLIP_DURATION_SECONDS} sec. long`);
    if(video.duration > MAX_VIDEO_DURATION_SECONDS) throw new Error("Videos longer than 20 minutes are not supported");

    const thumbnail = video.thumbnail?.trim()

    return {
        videoId: video.id,
        title: video.track?.trim() || video.title.trim(),
        artist: video.artist?.trim() || video.uploader?.trim() || video.channel?.trim() || "Unknown artist",
        duration: video.duration,
        ...(thumbnail ? { thumbnail } : {})
    }
}

async function cleanDownloadArtifacts(outputPath: string){
    const directory = path.dirname(outputPath);
    const basename = path.parse(outputPath).name;
    let entries: string[];

    try{
        entries = await fs.readdir(directory)
    }catch {
        return
    };

    await Promise.all(
        entries
            .filter((entry) => entry.startsWith(`${basename}.`))
            .map((entry) => 
                fs.rm(path.join(directory, entry), {
                    force: true,
                    recursive: true
                })
            )
    )
}

export async function downloadYouTubeClip(url: string, outputPath: string, clipStartSeconds: number, signal?: AbortSignal): Promise<DownloadedYouTubeClip> {
    if (path.extname(outputPath).toLowerCase() !== ".mp3") throw new Error("The YouTube output path must end with .mp3");
    await fs.mkdir(path.dirname(outputPath), { recursive: true});
    await cleanDownloadArtifacts(outputPath);

    const downloadStart = Math.max(0, clipStartSeconds - 1);
    const downloadEnd = clipStartSeconds + CLIP_DURATION_SECONDS + 1;
    const localClipStartSeconds = clipStartSeconds - downloadStart;

    const parsedPath = path.parse(outputPath);
    const outputTemplate = path.join(parsedPath.dir, `${parsedPath.name}.%(ext)s`);
    try{
        await runProcess(
        YT_DLP_BIN,
        [
          "--ignore-config",
          "--no-playlist",
          "--max-downloads",
          "1",
          "--match-filter",
          "!is_live",
          "--no-progress",
          "--force-overwrites",
          "--format",
          "bestaudio/best",
          "--extract-audio",
          "--audio-format",
          "mp3",
          "--audio-quality",
          "2",
          "--download-sections",
          `*${downloadStart}-${downloadEnd}`,
          "--output",
          outputTemplate,
          "--",
          url
        ],
        signal,
      )
      const stats = await fs.stat(outputPath);
      if (!stats.isFile() || stats.size === 0) throw new Error("yt-dlp created an empty audio file");
      if (stats.size > MAX_DOWNLOAD_SIZE_BYTES) throw new Error("The downloaded audio exceeds 30 MB");

      return {
          originalPath: outputPath,
          localClipStartSeconds,
      }
    } catch (error) {
        await cleanDownloadArtifacts(outputPath);
        throw new Error("Could not download audio from YouTube", {
            cause: error
        })
    }
}
