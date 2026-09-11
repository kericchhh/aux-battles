export type ButtonVariant =
  | "primary"
  | "secondary"
  | "ghost";

const variants: Record<ButtonVariant, string> = {
  primary:
    "bg-primary text-black hover:bg-primary-hover",

  secondary:
    "border border-primary text-primary hover:bg-primary hover:text-black",

  ghost:
    "text-muted hover:bg-white/5 hover:text-foreground",
};

export function buttonStyles(
  variant: ButtonVariant = "primary",
) {
  return [
    "inline-flex items-center justify-center",
    "rounded-xl px-5 py-3 font-medium",
    "transition-colors",
    "focus-visible:outline-2",
    "focus-visible:outline-offset-2",
    "focus-visible:outline-primary",
    "disabled:cursor-not-allowed",
    "disabled:opacity-50",
    variants[variant],
  ].join(" ");
}
