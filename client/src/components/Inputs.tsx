import type { InputHTMLAttributes } from "react";

type InputProps = InputHTMLAttributes<HTMLInputElement>;

export default function Input({ className = "", ...props }: InputProps) {
  return (
    <input
      {...props}
      className={`min-h-10 w-full rounded-lg border border-white/12 bg-neutral-800 px-3 py-2.5 text-sm text-foreground outline-none transition-colors placeholder:text-muted/70 hover:border-white/20 focus:border-primary/70 focus:ring-2 focus:ring-primary/15 disabled:cursor-not-allowed disabled:opacity-60 ${className}`}
    />
  );
}
