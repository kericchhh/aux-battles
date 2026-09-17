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

function renderPanel(onReady = vi.fn()) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <SongUploadPanel onReady={onReady} />
    </QueryClientProvider>,
  );
  return onReady;
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
    const onReady = renderPanel();
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
      expect(onReady).toHaveBeenCalledWith({ id: "song-1", title: "Test Song" });
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
  });
});
