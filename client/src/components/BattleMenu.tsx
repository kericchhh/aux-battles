import { useState, type SyntheticEvent } from "react";
import { ArrowRight, Link2, Plus } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { createBattle, joinBattle } from "../api/battles";
import { errorMessage } from "../api/client";
import Button from "./Button";
import ErrorBanner from "./ErrorBanner";
import Input from "./Inputs";

type MenuTab = "create" | "join";

export default function BattleMenu() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<MenuTab>("create");
  const [rounds, setRounds] = useState("5");
  const [inviteCode, setInviteCode] = useState("");
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleCreate(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const roundCount = Number(rounds);
    if (!Number.isInteger(roundCount) || roundCount < 1 || roundCount > 10) {
      setError("Enter a whole number between 1 and 10.");
      return;
    }

    setIsSubmitting(true);
    try {
      const battle = await createBattle(roundCount);
      navigate(`/battle/${battle.id}`, { state: { battle } });
    } catch (caughtError) {
      setError(errorMessage(caughtError));
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleJoin(event: SyntheticEvent<HTMLFormElement>) {
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
      navigate(`/battle/${battle.id}`, { state: { battle } });
    } catch (caughtError) {
      setError(errorMessage(caughtError));
    } finally {
      setIsSubmitting(false);
    }
  }

  function switchTab(tab: MenuTab) {
    setActiveTab(tab);
    setError("");
  }

  return (
    <section aria-labelledby="battle-menu-title" className="overflow-hidden rounded-xl border border-white/10 bg-surface">
      <div className="border-b border-white/8 px-6 pb-5 pt-7 sm:px-8">
        <p className="text-sm font-medium text-primary">Jump right in</p>
        <h2 id="battle-menu-title" className="mt-1 text-2xl font-bold">Start a battle</h2>
      </div>

      <div role="tablist" aria-label="Battle options" className="grid grid-cols-2 gap-2 p-2">
        {(["create", "join"] as const).map((tab) => (
          <button
            key={tab}
            id={`${tab}-battle-tab`}
            type="button"
            role="tab"
            aria-selected={activeTab === tab}
            aria-controls={`${tab}-battle-panel`}
            onClick={() => switchTab(tab)}
            className={`min-h-10 rounded-lg px-4 text-sm font-medium transition-colors ${activeTab === tab ? "bg-white/10 text-foreground" : "text-muted hover:bg-white/5 hover:text-foreground"}`}
          >
            {tab === "create" ? <Plus aria-hidden="true" className="mr-2 inline size-4" /> : <Link2 aria-hidden="true" className="mr-2 inline size-4" />}
            {tab === "create" ? "Create battle" : "Join with code"}
          </button>
        ))}
      </div>

      <div className="min-h-72 px-6 pb-8 pt-5 sm:px-8">
        <form id="create-battle-panel" role="tabpanel" aria-labelledby="create-battle-tab" hidden={activeTab !== "create"} onSubmit={handleCreate} className="space-y-5">
          <div>
            <label htmlFor="rounds" className="mb-2 block text-sm font-medium">Number of rounds</label>
            <Input id="rounds" type="number" inputMode="numeric" min={1} max={10} value={rounds} onChange={(event) => setRounds(event.target.value)} aria-describedby="rounds-help" />
            <p id="rounds-help" className="mt-2 text-sm text-muted">Choose between 1 and 10 rounds.</p>
          </div>
          {error && <ErrorBanner message={error} />}
          <Button type="submit" loading={isSubmitting} loadingText="Creating battle…" className="w-full">
            Create battle <ArrowRight aria-hidden="true" className="ml-2 size-4" />
          </Button>
        </form>

        <form id="join-battle-panel" role="tabpanel" aria-labelledby="join-battle-tab" hidden={activeTab !== "join"} onSubmit={handleJoin} className="space-y-5">
          <div>
            <label htmlFor="invite-code" className="mb-2 block text-sm font-medium">Six-character invite code</label>
            <Input id="invite-code" type="text" value={inviteCode} onChange={(event) => setInviteCode(event.target.value.toUpperCase())} maxLength={6} autoCapitalize="characters" autoComplete="off" placeholder="A1B2C3" className="text-center font-mono text-xl uppercase tracking-[0.28em]" />
            <p className="mt-2 text-sm text-muted">Ask the host to share their room code.</p>
          </div>
          {error && <ErrorBanner message={error} />}
          <Button type="submit" loading={isSubmitting} loadingText="Joining battle…" className="w-full">
            Join battle <ArrowRight aria-hidden="true" className="ml-2 size-4" />
          </Button>
        </form>
      </div>
    </section>
  );
}
