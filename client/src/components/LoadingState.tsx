import type { ReactNode } from "react";

export default function LoadingState({ children = "Loading…" }: { children?: ReactNode }) {
  return (
    <div role="status" className="mx-auto w-full max-w-4xl animate-pulse p-4 sm:p-8">
      <span className="sr-only">{children}</span>
      <div aria-hidden="true" className="space-y-5">
        <div className="h-5 w-32 rounded-full bg-white/10" />
        <div className="h-24 rounded-2xl bg-white/8" />
        <div className="grid grid-cols-2 gap-4">
          <div className="h-28 rounded-2xl bg-white/8" />
          <div className="h-28 rounded-2xl bg-white/8" />
        </div>
        <div className="h-64 rounded-3xl bg-white/8" />
      </div>
    </div>
  );
}
