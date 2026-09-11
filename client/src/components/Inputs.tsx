import type { InputHTMLAttributes } from "react";

type InputProps = InputHTMLAttributes<HTMLInputElement>;

export default function Input({ className = "", ...props }: InputProps) {
  return (
    <input
      {...props}
      className={`rounded bg-neutral-800 px-3 py-2 text-muted outline-none focus:ring-2 ${className}`}
    />
  );
}
