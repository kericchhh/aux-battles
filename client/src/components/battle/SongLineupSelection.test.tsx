import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getSongs } from "@/api/songs";
import type { BattleGame } from "@/hooks/useBattle";
import type { BattleView } from "@/lib/types/battle";
import SongLineupSelection from "./SongLineupSelection";

const uploadPanelMock = vi.hoisted(() => ({
  targetRound: 0,
  onReady: null as null | ((song: { id: string; title: string; artist: string }, targetRound: number) => void),
}));

vi.mock("@/api/songs", () => ({
  SONG_PAGE_SIZE: 20,
  getSongs: vi.fn(),
}));

vi.mock("./SongUploadPanel", () => ({
  default: ({ targetRound, onReady }: {
    targetRound: number;
    onReady: (song: { id: string; title: string; artist: string }, targetRound: number) => void;
  }) => (
    <div>
      <button type="button" onClick={() => {
        uploadPanelMock.targetRound = targetRound;
        uploadPanelMock.onReady = onReady;
      }}>Begin upload</button>
      <button type="button" onClick={() => uploadPanelMock.onReady?.(
        { id: "uploaded-song", title: "Uploaded Song", artist: "Uploaded Artist" },
        uploadPanelMock.targetRound,
      )}>Finish upload</button>
    </div>
  ),
}));

const songsMock = vi.mocked(getSongs);

const view: BattleView = {
  id: "battle-1",
  status: "SELECTING",
  inviteCode: "ABC123",
  currentRound: 1,
  rounds: 2,
  opponentJoined: true,
  myLineupLocked: false,
  opponentLineupLocked: false,
  outcome: null,
  myScore: 0,
  opponentScore: 0,
  round: null,
  previousRound: null,
};

function renderSelection(viewOverrides: Partial<BattleView> = {}) {
  const mutate = vi.fn();
  const game = {
    lineup: {
      mutate,
      isPending: false,
      error: null,
    },
  } as unknown as BattleGame;
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  render(
    <QueryClientProvider client={client}>
      <SongLineupSelection game={game} view={{ ...view, ...viewOverrides }} />
    </QueryClientProvider>,
  );

  return mutate;
}

describe("SongLineupSelection", () => {
  beforeEach(() => {
    songsMock.mockReset();
    songsMock.mockResolvedValue([
      { id: "song-1", title: "First Song", artist: "First Artist", genre: "Rock", duration: 120 },
      { id: "song-2", title: "Second Song", artist: "Second Artist", genre: "Pop", duration: 150 },
    ]);
    uploadPanelMock.targetRound = 0;
    uploadPanelMock.onReady = null;
  });

  it("submits one distinct song for every round", async () => {
    const user = userEvent.setup();
    const mutate = renderSelection();

    await user.click(await screen.findByRole("button", { name: /First Song/ }));
    await user.click(screen.getByRole("button", { name: /Second Song/ }));
    await user.click(screen.getByRole("button", { name: "Lock in lineup" }));

    expect(mutate).toHaveBeenCalledWith(["song-1", "song-2"]);
  });

  it("shows a waiting state after the lineup is locked", () => {
    renderSelection({ myLineupLocked: true });

    expect(screen.getByRole("status")).toHaveTextContent(
      "Waiting for your opponent to lock their lineup",
    );
    expect(screen.queryByRole("button", { name: "Lock in lineup" })).not.toBeInTheDocument();
  });

  it("keeps a completed upload in the round where it began", async () => {
    const user = userEvent.setup();
    renderSelection();

    await user.click(screen.getByRole("button", { name: "Begin upload" }));
    await user.click(screen.getByRole("button", { name: /Round 2Select a song/ }));
    await user.click(screen.getByRole("button", { name: "Finish upload" }));

    expect(screen.getByRole("button", { name: /Round 1Uploaded Song/ })).toBeVisible();
    expect(screen.getByRole("button", { name: /Round 2Select a song/ })).toHaveAttribute("aria-pressed", "true");
  });
});
