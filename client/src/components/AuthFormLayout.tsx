interface AuthFormLayoutProps {
  title: string;
  onSubmit: (e: React.FormEvent<HTMLFormElement>) => void;
  children: React.ReactNode;
}

export default function AuthFormLayout({
  title,
  onSubmit,
  children,
}: AuthFormLayoutProps) {
  return (
    <main className="bg-app grid min-h-dvh w-full place-items-center px-4 py-10 text-foreground">
      <div className="w-full max-w-md">
        <a href="/" className="mb-7 flex items-center justify-center gap-2 text-lg font-bold tracking-tight">
          <span className="text-primary">AUX</span> BATTLES
        </a>
        <form onSubmit={onSubmit} className="flex w-full flex-col gap-5 rounded-xl border border-white/10 bg-surface p-6 sm:p-8">
          <div className="mb-2">
            <p className="mb-2 text-sm font-semibold uppercase tracking-[0.18em] text-primary">Welcome</p>
            <h1 className="text-3xl font-bold text-foreground">{title}</h1>
          </div>

          {children}
        </form>
      </div>
    </main>
  );
}
