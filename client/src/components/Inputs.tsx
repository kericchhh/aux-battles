interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

export default function Input(props: InputProps) {
  return (
    <input
      {...props}
      className="bg-neutral-800 text-muted  rounded px-3 py-2 outline-none focus:ring-2"
    />
  );
}
