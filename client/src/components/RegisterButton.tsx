interface RegisterButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  loading?: boolean;
}

export default function RegisterButton({ loading, children, disabled, ...props }: RegisterButtonProps) {
  return (
    <button
      {...props}
      disabled={disabled || loading}
      className="bg-[#5964a6] hover:bg-[#959cc6] disabled:opacity-50 text-[#f4f0f0]  font-semibold rounded-xl px-3 py-2 mt-2"
    >
      {children}
    </button>
  );
}
