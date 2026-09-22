import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  getSongStatus,
  getYouTubeInfo,
  importYouTubeSong,
  uploadSong,
} from "@/api/songs";
import SongUploadPanel from "./SongUploadPanel";

vi.mock("@/api/songs", () => ({
  uploadSong: vi.fn(),
  getSongStatus: vi.fn(),
  getYouTubeInfo: vi.fn(),
  importYouTubeSong: vi.fn(),
}));

const uploadMock = vi.mocked(uploadSong);
const statusMock = vi.mocked(getSongStatus);
const youtubeInfoMock = vi.mocked(getYouTubeInfo);
const youtubeImportMock = vi.mocked(importYouTubeSong);

function renderPanel(onReady = vi.fn(), targetRound = 0) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const view = render(
    <QueryClientProvider client={client}>
      <SongUploadPanel targetRound={targetRound} onReady={onReady} />
    </QueryClientProvider>,
  );
  return {
    onReady,
    rerenderTarget(nextTargetRound: number) {
      view.rerender(
        <QueryClientProvider client={client}>
          <SongUploadPanel targetRound={nextTargetRound} onReady={onReady} />
        </QueryClientProvider>,
      );
    },
  };
}

describe("SongUploadPanel", () => {
  beforeEach(() => {
    uploadMock.mockReset();
    statusMock.mockReset();
    youtubeInfoMock.mockReset();
    youtubeImportMock.mockReset();
    Object.defineProperty(URL, "createObjectURL", {
      configurable: true,
      value: vi.fn(() => "blob:test-song"),
    });
    Object.defineProperty(URL, "revokeObjectURL", {
      configurable: true,
      value: vi.fn(),
    });
  });

  it("uploads a selected clip and reports the song when processing completes", async () => {
    const user = userEvent.setup();
    const { onReady } = renderPanel();
    uploadMock.mockResolvedValue({ id: "song-1", status: "PROCESSING" });
    statusMock.mockResolvedValue({
      id: "song-1",
      title: "Test Song",
      artist: "Test Artist",
      status: "READY",
      processingError: null,
      workerAvailable: null,
    });

    const file = new File(["audio"], "test.mp3", { type: "audio/mpeg" });
    await user.upload(screen.getByLabelText("Upload MP3 file"), file);

    const audio = document.querySelector("audio");
    expect(audio).not.toBeNull();
    Object.defineProperty(audio!, "duration", { configurable: true, value: 60 });
    fireEvent.loadedMetadata(audio!);

    await user.type(screen.getByPlaceholderText("Title"), "Test Song");
    await user.type(screen.getByPlaceholderText("Artist"), "Test Artist");
    await user.type(screen.getByPlaceholderText("Genre"), "Rock");
    fireEvent.change(screen.getByRole("slider"), { target: { value: "15" } });
    await user.click(screen.getByRole("button", { name: "Upload and process" }));

    await waitFor(() => {
      expect(uploadMock).toHaveBeenCalledWith(expect.objectContaining({
        file,
        title: "Test Song",
        artist: "Test Artist",
        genre: "Rock",
        clipStartSeconds: 15,
      }), expect.any(Object));
      expect(onReady).toHaveBeenCalledWith(
        { id: "song-1", title: "Test Song", artist: "Test Artist" },
        0,
      );
    });

    expect(screen.getByText("Song ready and selected.")).toBeVisible();
  });

  it("explains when an uploaded song is queued without an available worker", async () => {
    const user = userEvent.setup();
    renderPanel();
    uploadMock.mockResolvedValue({ id: "song-2", status: "PROCESSING" });
    statusMock.mockResolvedValue({
      id: "song-2",
      title: "Queued Song",
      artist: "Test Artist",
      status: "PROCESSING",
      processingError: null,
      workerAvailable: false,
    });

    const file = new File(["audio"], "queued.mp3", { type: "audio/mpeg" });
    await user.upload(screen.getByLabelText("Upload MP3 file"), file);
    const audio = document.querySelector("audio");
    Object.defineProperty(audio!, "duration", { configurable: true, value: 60 });
    fireEvent.loadedMetadata(audio!);
    await user.type(screen.getByPlaceholderText("Title"), "Queued Song");
    await user.type(screen.getByPlaceholderText("Artist"), "Test Artist");
    await user.type(screen.getByPlaceholderText("Genre"), "Rock");
    await user.click(screen.getByRole("button", { name: "Upload and process" }));

    expect(await screen.findByText(/processing service is currently offline/i)).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Choose from catalog instead" }));
    expect(screen.getByLabelText("Upload MP3 file")).toBeEnabled();
    expect(screen.queryByText(/processing service is currently offline/i)).not.toBeInTheDocument();
  });

  it("keeps the round selected when the upload began", async () => {
    const user = userEvent.setup();
    let finishProcessing!: (value: Awaited<ReturnType<typeof getSongStatus>>) => void;
    uploadMock.mockResolvedValue({ id: "song-3", status: "PROCESSING" });
    statusMock.mockImplementation(() => new Promise((resolve) => {
      finishProcessing = resolve;
    }));
    const { onReady, rerenderTarget } = renderPanel(vi.fn(), 0);

    const file = new File(["audio"], "pinned.mp3", { type: "audio/mpeg" });
    await user.upload(screen.getByLabelText("Upload MP3 file"), file);
    const audio = document.querySelector("audio");
    Object.defineProperty(audio!, "duration", { configurable: true, value: 60 });
    fireEvent.loadedMetadata(audio!);
    await user.type(screen.getByPlaceholderText("Title"), "Pinned Song");
    await user.type(screen.getByPlaceholderText("Artist"), "Pinned Artist");
    await user.type(screen.getByPlaceholderText("Genre"), "Rock");
    await user.click(screen.getByRole("button", { name: "Upload and process" }));
    await waitFor(() => expect(statusMock).toHaveBeenCalled());

    rerenderTarget(1);
    finishProcessing({
      id: "song-3",
      title: "Pinned Song",
      artist: "Pinned Artist",
      status: "READY",
      processingError: null,
      workerAvailable: null,
    });

    await waitFor(() => expect(onReady).toHaveBeenCalledWith(
      { id: "song-3", title: "Pinned Song", artist: "Pinned Artist" },
      0,
    ));
  });

  it("imports a selected YouTube clip and reports it when processing completes", async () => {
    const user = userEvent.setup();
    const { onReady } = renderPanel();
    youtubeInfoMock.mockResolvedValue({
      videoId: "video-1",
      title: "YouTube Song",
      artist: "YouTube Artist",
      duration: 180,
      thumbnail: "https://i.ytimg.com/test.jpg",
    });
    youtubeImportMock.mockResolvedValue({ id: "song-youtube", status: "PROCESSING" });
    statusMock.mockResolvedValue({
      id: "song-youtube",
      title: "YouTube Song",
      artist: "YouTube Artist",
      status: "READY",
      processingError: null,
      workerAvailable: null,
    });

    await user.click(screen.getByRole("tab", { name: "YouTube URL" }));
    await user.type(
      screen.getByRole("textbox", { name: "YouTube URL" }),
      "https://www.youtube.com/watch?v=video-1",
    );
    await user.click(screen.getByRole("button", { name: "Load video" }));

    expect(await screen.findByDisplayValue("YouTube Song")).toBeVisible();
    expect(youtubeInfoMock).toHaveBeenCalledWith(
      "https://www.youtube.com/watch?v=video-1",
      expect.any(Object),
    );

    await user.type(screen.getByPlaceholderText("Genre"), "Rock");
    fireEvent.change(screen.getByRole("slider"), { target: { value: "30" } });
    await user.click(screen.getByRole("button", { name: "Import and process" }));

    await waitFor(() => {
      expect(youtubeImportMock).toHaveBeenCalledWith(expect.objectContaining({
        youtubeUrl: "https://www.youtube.com/watch?v=video-1",
        title: "YouTube Song",
        artist: "YouTube Artist",
        genre: "Rock",
        clipStartSeconds: 30,
      }), expect.any(Object));
      expect(onReady).toHaveBeenCalledWith(
        { id: "song-youtube", title: "YouTube Song", artist: "YouTube Artist" },
        0,
      );
    });
  });
});
