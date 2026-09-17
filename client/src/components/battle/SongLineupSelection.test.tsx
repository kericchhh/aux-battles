import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BattleView } from "../../api/battles";
import { getSongs } from "../../api/songs";
import type { BattleGame } from "../../hooks/useBattle";
import SongLineupSelection from "./SongLineupSelection";

vi.mock("../../api/songs", () => ({
  SONG_PAGE_SIZE: 20,
  getSongs: vi.fn(),
}));

vi.mock("./SongUploadPanel", () => ({
  default: () => <div>Upload panel</div>,
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
});
