import type { HTMLAttributes } from "react";

export default function Panel({ className = "", ...props }: HTMLAttributes<HTMLElement>) {
  return (
    <section
      {...props}
      className={`rounded-xl border border-white/10 bg-surface p-5 sm:p-6 ${className}`}
    />
  );
}
