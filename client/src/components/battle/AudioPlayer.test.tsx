import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BattleRound } from "../../api/battles";
import AudioPlayer from "./AudioPlayer";

const waveMock = vi.hoisted(() => ({
  listeners: new Map<string, (...args: number[]) => void>(),
  load: vi.fn(),
  destroy: vi.fn(),
  setMuted: vi.fn(),
  playPause: vi.fn(),
}));

vi.mock("wavesurfer.js", () => ({
  default: {
    create: vi.fn(() => ({
      load: waveMock.load,
      destroy: waveMock.destroy,
      setMuted: waveMock.setMuted,
      playPause: waveMock.playPause,
      on: (event: string, listener: (...args: number[]) => void) => {
        waveMock.listeners.set(event, listener);
        return vi.fn();
      },
    })),
  },
}));

const round: BattleRound = {
  id: "round-1",
  number: 1,
  status: "GUESSING",
  myStage: "BASS",
  myPoints: 0,
  myFinished: false,
  opponentFinished: false,
  myAttempts: 1,
};

describe("AudioPlayer", () => {
  beforeEach(() => {
    waveMock.listeners.clear();
    waveMock.load.mockClear();
  });

  it("does not reload WaveSurfer during an ordinary render", () => {
    const { rerender } = render(<AudioPlayer round={round} />);
    rerender(<AudioPlayer round={round} />);
    expect(waveMock.load).toHaveBeenCalledTimes(1);
  });

  it("enables labeled playback and volume controls when ready", async () => {
    const user = userEvent.setup();
    render(<AudioPlayer round={round} />);
    act(() => waveMock.listeners.get("ready")?.(12));

    expect(screen.getByRole("button", { name: "Play audio clue" })).toBeEnabled();
    expect(screen.getByLabelText("Interactive audio waveform")).toBeVisible();
    expect(screen.getByText("0:12")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Mute audio" }));
    expect(waveMock.setMuted).toHaveBeenCalledWith(true);
    expect(screen.getByRole("button", { name: "Unmute audio" })).toBeVisible();
  });

  it("shows a recoverable error when audio loading fails", async () => {
    const user = userEvent.setup();
    render(<AudioPlayer round={round} />);
    act(() => waveMock.listeners.get("error")?.());

    expect(screen.getByRole("alert")).toHaveTextContent("Audio could not be loaded");
    await user.click(screen.getByRole("button", { name: "Retry audio" }));
    expect(waveMock.load).toHaveBeenCalledTimes(2);
  });
});
