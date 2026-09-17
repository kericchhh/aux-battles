import { ArrowLeft, Disc3 } from "lucide-react";
import { buttonStyles } from "./buttonStyles";

interface ErrorPageProps {
  code: string;
  title: string;
  description: string;
  actionHref: string;
  actionLabel: string;
  secondaryAction?: { label: string; onClick: () => void };
}

export default function ErrorPage({ code, title, description, actionHref, actionLabel, secondaryAction }: ErrorPageProps) {
  return (
    <main className="bg-app grid min-h-dvh place-items-center px-4 py-12 text-foreground">
      <section className="w-full max-w-xl text-center">
        <div className="relative mx-auto mb-8 grid size-32 place-items-center rounded-full border border-primary/30 bg-primary/10">
          <div className="absolute inset-3 rounded-full border border-dashed border-primary/40" />
          <Disc3 aria-hidden="true" className="size-12 text-primary" />
        </div>
        <p className="mb-3 text-sm font-semibold uppercase tracking-[0.24em] text-primary">{code}</p>
        <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">{title}</h1>
        <p className="mx-auto mt-4 max-w-md text-muted">{description}</p>
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <a href={actionHref} className={buttonStyles()}><ArrowLeft aria-hidden="true" className="mr-2 size-4" />{actionLabel}</a>
          {secondaryAction && <button type="button" className={buttonStyles("secondary")} onClick={secondaryAction.onClick}>{secondaryAction.label}</button>}
        </div>
      </section>
    </main>
  );
}
