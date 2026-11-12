export function ScoreLine({ mine, theirs }: { mine: number; theirs: number }) {
  return (
    <span className="font-mono tabular">
      <span aria-hidden>
        {mine}–{theirs}
      </span>
      <span className="sr-only">
        You {mine}, them {theirs}
      </span>
    </span>
  )
}
