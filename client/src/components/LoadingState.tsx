import type { ReactNode } from "react";

export default function LoadingState({ children = "Loading…" }: { children?: ReactNode }) {
  return (
    <p role="status" className="p-8 text-muted">
      {children}
    </p>
  );
}
