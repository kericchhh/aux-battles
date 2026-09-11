import type { HTMLAttributes } from "react";

export default function PageShell({
  className = "",
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      {...props}
      className={`
        bg-app
        min-h-[calc(100dvh-4rem)]
        w-full
        px-4 py-8
        text-foreground
        ${className}
      `}
    />
  );
}
