import { useCallback, useEffect, useState } from "react";
import { Link, useLocation, useParams } from "react-router-dom";
import {
  getBattle,
  type Battle as BattleData,
  type BattleState,
} from "../api/battles";
import ErrorBanner from "../components/ErrorBanner";
import { useAuth } from "../context/AuthContext";
import { useSocket } from "../hooks/useSocket";

interface BattleLocationState {
  battle?: BattleData;
}

export default function Battle() {
  const { battleId } = useParams<{ battleId: string }>();
  const location = useLocation();
  const { userId } = useAuth();

  const initialBattle = (
    location.state as BattleLocationState | null
  )?.battle;

  const [battleState, setBattleState] =
    useState<BattleState | null>(() =>
      initialBattle
        ? { battle: initialBattle, round: null }
        : null,
    );

  const [isLoading, setIsLoading] = useState(!initialBattle);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  const socket = useSocket(battleId);

  const loadBattle = useCallback(
    async (showLoading = false) => {
      if (!battleId) {
        setError("Invalid battle URL.");
        setIsLoading(false);
        return;
      }

      if (showLoading) {
        setIsLoading(true);
      }

      try {
        const state = await getBattle(battleId);

        setBattleState(state);
        setError("");
      } catch (error) {
        setError(
          error instanceof Error
            ? error.message
            : "Could not load the battle.",
        );
      } finally {
        setIsLoading(false);
      }
    },
    [battleId],
  );

  useEffect(() => {
    void loadBattle(!initialBattle);
  }, [initialBattle, loadBattle]);

  useEffect(() => {
    if (!socket) return;

    const refreshBattle = () => {
      void loadBattle();
    };

    socket.on("battle:update", refreshBattle);
    socket.on("round:update", refreshBattle);

    return () => {
      socket.off("battle:update", refreshBattle);
      socket.off("round:update", refreshBattle);
    };
  }, [loadBattle, socket]);

  const copyInviteCode = async () => {
    const code = battleState?.battle.inviteCode;

    if (!code) return;

    try {
      await navigator.clipboard.writeText(code);

      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setError("Could not copy the invite code.");
    }
  };

  if (isLoading) {
    return (
      <main className="flex h-full items-center justify-center bg-[radial-gradient(circle_at_35%_35%,#29345f_0%,#171b32_35%,#0d1020_100%)] text-[#f4f0f7]">
        <p className="animate-pulse text-lg">
          Loading battle...
        </p>
      </main>
    );
  }

  if (!battleState) {
    return (
      <main className="flex h-full items-center justify-center bg-[#0d1020] px-6 text-[#f4f0f7]">
        <div className="w-full max-w-md space-y-5 text-center">
          <ErrorBanner
            message={error || "Battle not found."}
          />

          <Link
            to="/"
            className="inline-block rounded-xl bg-[#5964a6] px-6 py-3 transition-colors hover:bg-[#959cc6] hover:text-black"
          >
            Return to lobby
          </Link>
        </div>
      </main>
    );
  }

  const { battle, round } = battleState;

  const isHost = battle.hostId === userId;
  const waitingForOpponent =
    battle.status === "PENDING" && !battle.guestId;

  return (
    <main className="h-full overflow-y-auto bg-[radial-gradient(circle_at_35%_35%,#29345f_0%,#171b32_35%,#0d1020_100%)] px-6 py-10 text-[#f4f0f7]">
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
        {error && <ErrorBanner message={error} />}

        <header className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-sm uppercase tracking-[0.25em] text-[#959cc6]">
              Aux Battle
            </p>

            <h1 className="mt-1 text-3xl font-semibold">
              Round {battle.currentRound} of {battle.rounds}
            </h1>
          </div>

          <span className="rounded-full border border-[#5964a6] bg-[#090a11]/70 px-4 py-2 text-sm">
            {battle.status}
          </span>
        </header>

        {waitingForOpponent ? (
          <section className="flex min-h-[380px] flex-col items-center justify-center rounded-2xl border border-[#5964a6] bg-[#090a11]/90 p-8 text-center shadow-xl">
            <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-[#5964a6]/30">
              <span className="text-3xl">♫</span>
            </div>

            <h2 className="text-2xl font-semibold">
              Waiting for an opponent
            </h2>

            <p className="mt-2 max-w-md text-[#b9bdd8]">
              Share this code with the person you want to
              challenge. This page will update automatically when
              they join.
            </p>

            <button
              type="button"
              onClick={copyInviteCode}
              className="mt-8 rounded-xl border border-[#959cc6] bg-[#15182a] px-8 py-5 font-mono text-3xl font-bold tracking-[0.35em] transition-colors hover:bg-[#5964a6]"
            >
              {battle.inviteCode}
            </button>

            <p className="mt-3 text-sm text-[#959cc6]">
              {copied
                ? "Copied!"
                : "Click the code to copy"}
            </p>
          </section>
        ) : (
          <>
            <section className="grid gap-4 sm:grid-cols-2">
              <PlayerCard
                label="Host"
                active={isHost}
                connected
              />

              <PlayerCard
                label="Guest"
                active={!isHost}
                connected={Boolean(battle.guestId)}
              />
            </section>

            <section className="rounded-2xl border border-[#5964a6] bg-[#090a11]/90 p-8 text-center shadow-xl">
              {battle.status === "FINISHED" ? (
                <>
                  <h2 className="text-2xl font-semibold">
                    Battle finished
                  </h2>

                  <p className="mt-3 text-[#b9bdd8]">
                    {battle.winnerId
                      ? battle.winnerId === userId
                        ? "You won!"
                        : "Your opponent won."
                      : "The battle ended in a draw."}
                  </p>
                </>
              ) : round ? (
                <>
                  <p className="text-sm uppercase tracking-[0.2em] text-[#959cc6]">
                    Round in progress
                  </p>

                  <h2 className="mt-2 text-2xl font-semibold">
                    Guess your opponent&apos;s song
                  </h2>

                  <p className="mt-3 text-[#b9bdd8]">
                    The audio player and guessing controls are the
                    next step.
                  </p>
                </>
              ) : (
                <>
                  <p className="text-sm uppercase tracking-[0.2em] text-[#959cc6]">
                    Opponent connected
                  </p>

                  <h2 className="mt-2 text-2xl font-semibold">
                    Choose a song for this round
                  </h2>

                  <p className="mt-3 text-[#b9bdd8]">
                    Song selection is ready to be connected next.
                  </p>
                </>
              )}
            </section>
          </>
        )}
      </div>
    </main>
  );
}

interface PlayerCardProps {
  label: string;
  active: boolean;
  connected: boolean;
}

function PlayerCard({
  label,
  active,
  connected,
}: PlayerCardProps) {
  return (
    <div
      className={`rounded-2xl border p-5 ${
        active
          ? "border-[#959cc6] bg-[#5964a6]/25"
          : "border-[#343958] bg-[#090a11]/80"
      }`}
    >
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-[#959cc6]">
            {label}
          </p>

          <p className="mt-1 font-medium">
            {active ? "You" : "Opponent"}
          </p>
        </div>

        <span
          className={`h-3 w-3 rounded-full ${
            connected
              ? "bg-emerald-400"
              : "bg-gray-500"
          }`}
        />
      </div>
    </div>
  );
}
