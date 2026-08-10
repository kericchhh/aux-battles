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
    <div className="min-h-screen w-full flex items-center justify-center bg-[radial-gradient(circle_at_35%_35%,#29345f_0%,#171b32_35%,#0d1020_100%)] x-6">
      <form
        onSubmit={onSubmit}
        className="
          w-full max-w-md
          flex flex-col gap-5
          rounded-3xl
          bg-[#090a11]
          p-8
          shadow-2xl
          border border-white/5
        "
      >
        <h1 className="text-3xl font-bold text-[#f4f0f7] mb-2">
          {title}
        </h1>

        {children}
      </form>
    </div>
  );
}
