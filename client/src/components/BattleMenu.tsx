import { useState, type SyntheticEvent } from "react";
import { useNavigate } from "react-router-dom";
import { createBattle, joinBattle } from "../api/battles";
import ErrorBanner from "./ErrorBanner";

export default function BattleMenu() {
  const navigate = useNavigate();

  const [activeTab, setActiveTab] = useState<"create" | "join">(
    "create"
  );
  const [rounds, setRounds] = useState("");
  const [inviteCode, setInviteCode] = useState("");
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleCreate = async (
    event: SyntheticEvent<HTMLFormElement>
  ) => {
    event.preventDefault();
    setError("");

    const roundCount = Number(rounds);

    if (
      !Number.isInteger(roundCount) ||
      roundCount < 1 ||
      roundCount > 10
    ) {
      setError("Enter a whole number between 1 and 10.");
      return;
    }

    setIsSubmitting(true);

    try {
      const battle = await createBattle(roundCount);

      navigate(`/battle/${battle.id}`, {
        state: { battle },
      });
    } catch (error) {
      setError(
        error instanceof Error
          ? error.message
          : "Could not create battle."
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleJoin = async (
    event: SyntheticEvent<HTMLFormElement>
  ) => {
    event.preventDefault();
    setError("");

    const normalizedCode = inviteCode.trim().toUpperCase();

    if (normalizedCode.length !== 6) {
      setError("Invite codes must be 6 characters long.");
      return;
    }

    setIsSubmitting(true);

    try {
      const battle = await joinBattle(normalizedCode);

      navigate(`/battle/${battle.id}`, {
        state: { battle },
      });
    } catch (error) {
      setError(
        error instanceof Error
          ? error.message
          : "Could not join battle."
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const switchTab = (tab: "create" | "join") => {
    setActiveTab(tab);
    setError("");
  };

  return (
    <div className="flex h-full w-full items-center justify-center">
      <div className="w-[450px] overflow-hidden rounded-2xl border border-primary bg-surface font-sans text-[#f4f0f7] shadow-lg backdrop-blur-sm">
        <div className="flex">
          <button
            type="button"
            onClick={() => switchTab("create")}
            className={`flex-1 py-4 text-center transition-colors ${
              activeTab === "create"
                ? "bg-primary font-semibold"
                : "hover:bg-primary-soft"
            }`}
          >
            Create a room
          </button>

          <button
            type="button"
            onClick={() => switchTab("join")}
            className={`flex-1 py-4 text-center transition-colors ${
              activeTab === "join"
                ? "bg-primary font-semibold"
                : "hover:bg-primary-soft"
            }`}
          >
            Join with a code
          </button>
        </div>

        <div className="flex min-h-[300px] flex-col items-center justify-center p-10">
          {activeTab === "create" && (
            <form
              onSubmit={handleCreate}
              className="flex w-full flex-col items-center space-y-6"
            >
              <input
                type="number"
                inputMode="numeric"
                value={rounds}
                onChange={(event) => setRounds(event.target.value)}
                placeholder="Number of rounds (1-10)"
                aria-label="Number of rounds"
                className="w-full rounded-xl border border-gray-300 bg-transparent px-4 py-4 text-center text-white placeholder-gray-400 transition-all focus:border-white focus:outline-none focus:ring-1 focus:ring-white"
              />

              {error && (
                <div className="w-full">
                  <ErrorBanner message={error} />
                </div>
              )}

              <button
                type="submit"
                disabled={isSubmitting}
                className="rounded-xl bg-primary px-8 py-3 text-sm font-medium uppercase tracking-wider transition-colors hover:bg-primary-soft hover:text-black disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isSubmitting ? "Creating..." : "Create Battle"}
              </button>
            </form>
          )}

          {activeTab === "join" && (
            <form
              onSubmit={handleJoin}
              className="flex w-full flex-col items-center space-y-6"
            >
              <input
                type="text"
                value={inviteCode}
                onChange={(event) =>
                  setInviteCode(event.target.value.toUpperCase())
                }
                maxLength={6}
                autoCapitalize="characters"
                placeholder="Enter invite code"
                aria-label="Invite code"
                className="w-full rounded-xl border border-gray-300 bg-transparent px-4 py-4 text-center uppercase tracking-widest text-white placeholder:normal-case placeholder:tracking-normal placeholder-gray-400 transition-all focus:border-white focus:outline-none focus:ring-1 focus:ring-white"
              />

              {error && (
                <div className="w-full">
                  <ErrorBanner message={error} />
                </div>
              )}

              <button
                type="submit"
                disabled={isSubmitting}
                className="rounded-xl bg-primary px-12 py-3 text-sm font-medium tracking-wider transition-colors hover:bg-primary-soft hover:text-black disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isSubmitting ? "Joining..." : "Join"}
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}
