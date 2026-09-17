import { ArrowLeft, Radio } from "lucide-react";
import { Link } from "react-router-dom";

interface BattleHeaderProps {
  connection: string;
  currentRound: number;
  rounds: number;
}

export default function BattleHeader({ connection, currentRound, rounds }: BattleHeaderProps) {
  return (
    <header className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Link to="/" className="inline-flex items-center gap-2 rounded-lg text-sm text-muted transition hover:text-foreground">
          <ArrowLeft aria-hidden="true" className="size-4" /> Back to lobby
        </Link>
        <p role="status" className="inline-flex items-center gap-2 text-xs text-muted">
          <Radio aria-hidden="true" className="size-3.5 text-secondary" />{connection}
        </p>
      </div>

      <div aria-label={`Round ${currentRound} of ${rounds}`} className="rounded-xl border border-white/10 bg-surface px-5 py-4">
        <div className="mb-3 flex items-center justify-between gap-4">
          <h1 className="text-lg font-semibold">Round {currentRound} <span className="font-normal text-muted">of {rounds}</span></h1>
          <span className="text-xs uppercase tracking-[0.18em] text-muted">Battle progress</span>
        </div>
        <ol className="flex gap-2" aria-hidden="true">
          {Array.from({ length: rounds }, (_, index) => {
            const number = index + 1;
            return <li key={number} className={`h-1.5 flex-1 rounded-full ${number < currentRound ? "bg-primary/60" : number === currentRound ? "bg-primary" : "bg-white/10"}`} />;
          })}
        </ol>
      </div>
    </header>
  );
}
