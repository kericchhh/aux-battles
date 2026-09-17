import { Link } from "react-router-dom";
import type { BattleView } from "@/lib/types/battle";
import { buttonStyles } from "../buttonStyles";
import Panel from "../Panel";

type BattleResultProps = Pick<BattleView, "outcome" | "myScore" | "opponentScore">;

export default function BattleResult({ outcome, myScore, opponentScore }: BattleResultProps) {
  const title = outcome === "WIN" ? "You won!" : outcome === "DRAW" ? "It's a draw" : "Your opponent won";

  return (
    <Panel className="text-center">
      <h2 className="text-3xl font-semibold">{title}</h2>
      <p className="my-5 text-muted">
        Final score: {myScore} – {opponentScore}
      </p>
      <Link className={buttonStyles()} to="/">
        Start another battle
      </Link>
    </Panel>
  );
}
