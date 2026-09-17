import { useState, type FormEvent } from "react";
import { errorMessage } from "@/api/client";
import type { BattleGame } from "@/hooks/useBattle";
import type { BattleRound } from "@/lib/types/battle";
import Button from "../Button";
import ErrorBanner from "../ErrorBanner";
import Input from "../Inputs";
import Panel from "../Panel";
import AudioPlayer from "./AudioPlayer";

interface GuessingPanelProps {
  game: BattleGame;
  round: BattleRound;
}

export default function GuessingPanel({ game, round }: GuessingPanelProps) {
  const [guess, setGuess] = useState("");

  function submitGuess(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedGuess = guess.trim();

    if (!normalizedGuess || round.myFinished || game.guess.isPending) return;

    game.guess.mutate(
      {
        roundId: round.id,
        guess: normalizedGuess,
        expectedAttempt: round.myAttempts + 1,
      },
      { onSuccess: () => setGuess("") },
    );
  }

  const currentError = game.guess.variables?.roundId === round.id ? game.guess.error : null;
  const currentResult = game.guess.data?.roundId === round.id ? game.guess.data : null;

  return (
    <Panel className="mx-auto w-full max-w-3xl">
      <div className="text-center">
        <p className="text-xs font-semibold uppercase tracking-[0.22em] text-primary">Current guessing stage</p>
        <h2 className="mt-2 text-2xl font-semibold sm:text-3xl">Name that track</h2>
      </div>
      <p className="my-4 text-center text-sm text-muted">
        Stage: {round.myStage} · {round.myAttempts} of 4 attempts used
      </p>

      <AudioPlayer key={`${round.id}:${round.myStage}`} round={round} />

      {round.myFinished ? (
        <p role="status" className="mt-6 rounded-xl border border-white/10 bg-white/5 p-4 text-center">
          You've finished. {round.opponentFinished ? "Preparing the result…" : "Waiting for your opponent…"}
        </p>
      ) : (
        <form onSubmit={submitGuess} className="mx-auto mt-6 flex max-w-xl flex-col gap-3">
          <label htmlFor="guess" className="text-sm font-medium">Song title</label>
          <Input
            id="guess"
            autoComplete="off"
            maxLength={255}
            required
            placeholder="Enter the song title"
            value={guess}
            onChange={(event) => setGuess(event.target.value)}
          />
          <Button
            type="submit"
            loading={game.guess.isPending}
            loadingText="Checking…"
            disabled={!guess.trim()}
          >
            Submit guess
          </Button>
        </form>
      )}

      {currentError && (
        <div className="mt-4">
          <ErrorBanner message={errorMessage(currentError)} />
        </div>
      )}

      {currentResult && (
        <p role="status" className="mt-4">
          {currentResult.correct ? `Correct! +${currentResult.pointsAwarded} points.` : "Incorrect guess."}
        </p>
      )}
    </Panel>
  );
}
