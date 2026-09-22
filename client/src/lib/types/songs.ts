export type Song = {
  id: string;
  title: string;
  artist: string;
  genre: string;
  duration: number;
};

export type SongProcessingStatus = {
  id: string;
  title: string;
  artist: string;
  status: "PROCESSING" | "READY" | "FAILED";
  processingError: string | null;
  workerAvailable: boolean | null;
};

export type SongUpload = {
  file: File;
  title: string;
  artist: string;
  genre: string;
  album?: string;
  clipStartSeconds: number;
};

export interface YouTubeInfo {
  videoId: string;
  title: string;
  artist: string;
  duration: number;
  thumbnail?: string;
}

export interface YouTubeSongImport {
  youtubeUrl: string;
  title: string;
  artist: string;
  genre: string;
  album?: string;
  clipStartSeconds: number;
}
