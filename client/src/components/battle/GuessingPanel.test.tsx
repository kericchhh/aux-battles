import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { BattleGame } from "@/hooks/useBattle";
import type { BattleRound } from "@/lib/types/battle";
import GuessingPanel from "./GuessingPanel";

vi.mock("./AudioPlayer", () => ({
  default: () => <div data-testid="audio-player" />,
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

function createGame(guessOverrides: Record<string, unknown> = {}) {
  return {
    guess: {
      mutate: vi.fn(),
      isPending: false,
      variables: undefined,
      error: null,
      data: undefined,
      ...guessOverrides,
    },
  } as unknown as BattleGame;
}

describe("GuessingPanel", () => {
  it("trims and submits a guess with the next expected attempt", async () => {
    const user = userEvent.setup();
    const mutate = vi.fn();
    const game = createGame({ mutate });

    render(<GuessingPanel game={game} round={round} />);

    expect(screen.getByText("Stage: BASS · 1 of 4 attempts used")).toBeVisible();
    expect(screen.getByTestId("audio-player")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Song title"), "  Halo  ");
    await user.click(screen.getByRole("button", { name: "Submit guess" }));

    expect(mutate).toHaveBeenCalledWith(
      {
        roundId: "round-1",
        guess: "Halo",
        expectedAttempt: 2,
      },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it("hides the form after the player finishes", () => {
    const finishedRound = {
      ...round,
      myFinished: true,
      opponentFinished: false,
    };

    render(<GuessingPanel game={createGame()} round={finishedRound} />);

    expect(screen.queryByLabelText("Song title")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(
      "You've finished. Waiting for your opponent…",
    );
  });

  it("only shows results belonging to the current round", () => {
    const game = createGame({
      data: {
        roundId: "round-1",
        correct: true,
        pointsAwarded: 3,
        playerFinished: true,
        roundFinished: false,
      },
    });

    render(<GuessingPanel game={game} round={round} />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "Correct! +3 points.",
    );
  });
});
