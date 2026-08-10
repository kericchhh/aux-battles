export default function ErrorBanner({ message }: { message: string }) {
  return (
    <div className="bg-red-950 text-red-300 text-sm p-2 rounded">{message}</div>
  );
}
