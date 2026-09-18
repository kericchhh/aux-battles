import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getSongStatus, uploadSong } from "@/api/songs";
import SongUploadPanel from "./SongUploadPanel";

vi.mock("@/api/songs", () => ({
  uploadSong: vi.fn(),
  getSongStatus: vi.fn(),
}));

const uploadMock = vi.mocked(uploadSong);
const statusMock = vi.mocked(getSongStatus);

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
});
