import { Link } from "react-router-dom";

interface BattleHeaderProps {
  connection: string;
  currentRound: number;
  rounds: number;
}

export default function BattleHeader({ connection, currentRound, rounds }: BattleHeaderProps) {
  return (
    <header className="flex flex-wrap items-center justify-between gap-3">
      <div>
        <Link to="/" className="text-sm text-muted hover:text-foreground">
          ← Lobby
        </Link>
        <h1 className="mt-2 text-3xl font-semibold">
          Round {currentRound} of {rounds}
        </h1>
      </div>

      <p role="status" className="text-sm text-muted">
        {connection}
      </p>
    </header>
  );
}



