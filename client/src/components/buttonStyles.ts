export type ButtonVariant =
  | "primary"
  | "secondary"
  | "ghost";

const variants: Record<ButtonVariant, string> = {
  primary:
    "bg-foreground text-canvas hover:bg-primary/90",

  secondary:
    "border border-white/12 bg-transparent text-foreground hover:bg-white/5",

  ghost:
    "text-muted hover:bg-white/5 hover:text-foreground",
};

export function buttonStyles(
  variant: ButtonVariant = "primary",
) {
  return [
    "inline-flex items-center justify-center",
    "min-h-10 rounded-lg px-4 py-2.5 text-sm font-medium",
    "transition-colors duration-150",
    "focus-visible:outline-2",
    "focus-visible:outline-offset-2",
    "focus-visible:outline-primary",
    "disabled:cursor-not-allowed",
    "disabled:opacity-50",
    variants[variant],
  ].join(" ");
}
