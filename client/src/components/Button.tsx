import type { ButtonHTMLAttributes } from "react";
import {
  buttonStyles,
  type ButtonVariant,
} from "./buttonStyles";

interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  loading?: boolean;
  loadingText?: string;
}

export default function Button({
  variant = "primary",
  loading = false,
  loadingText = "Loading…",
  className = "",
  disabled,
  children,
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <button
      {...props}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading}
      className={`${buttonStyles(variant)} ${className}`}
    >
      {loading ? loadingText : children}
    </button>
  );
}
