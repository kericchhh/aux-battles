import { CircleAlert } from "lucide-react";

export default function ErrorBanner({ message }: { message: string }) {
  return (
    <div role="alert" className="flex items-start gap-3 rounded-xl border border-danger/25 bg-danger/10 p-4 text-sm text-red-100">
      <CircleAlert aria-hidden="true" className="mt-0.5 size-4 shrink-0 text-danger" />
      <span>{message}</span>
    </div>
  );
}
