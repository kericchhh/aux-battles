import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import BattleResult from "./BattleResult";
import ScoreBoard from "./ScoreBoard";

describe("BattleResult", () => {
  it.each([
    ["WIN", "You won!"],
    ["DRAW", "It's a draw"],
    ["LOSS", "Your opponent won"],
  ] as const)("renders the %s outcome", (outcome, title) => {
    render(
      <MemoryRouter>
        <BattleResult outcome={outcome} myScore={8} opponentScore={5} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("heading", { name: title })).toBeVisible();
    expect(screen.getByText("Final score: 8 – 5")).toBeVisible();
    expect(
      screen.getByRole("link", { name: "Start another battle" }),
    ).toHaveAttribute("href", "/");
  });
});

describe("ScoreBoard", () => {
  it("shows both scores and the opponent connection state", () => {
    render(
      <ScoreBoard
        myScore={4}
        opponentScore={2}
        opponentJoined
        previousRound={null}
      />,
    );

    expect(screen.getByText("Opponent · joined")).toBeVisible();
    expect(screen.getByText("4")).toBeVisible();
    expect(screen.getByText("2")).toBeVisible();
    expect(
      screen.queryByLabelText("Previous round result"),
    ).not.toBeInTheDocument();
  });

  it("renders the previous round breakdown when available", () => {
    render(
      <ScoreBoard
        myScore={7}
        opponentScore={6}
        opponentJoined={false}
        previousRound={{
          number: 2,
          myPoints: 3,
          opponentPoints: 1,
        }}
      />,
    );

    expect(screen.getByText("Opponent · waiting")).toBeVisible();
    const result = screen.getByLabelText("Previous round result");
    expect(result).toHaveTextContent("Round 2 result");
    expect(result).toHaveTextContent("You: +3 · Opponent: +1");
  });
});
