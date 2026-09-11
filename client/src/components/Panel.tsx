import type { HTMLAttributes } from "react";

export default function Panel({ className = "", ...props }: HTMLAttributes<HTMLElement>) {
  return (
    <section
      {...props}
      className={`rounded-2xl border border-primary/40 bg-surface/95 p-6 shadow-lg sm:p-8 ${className}`}
    />
  );
}
