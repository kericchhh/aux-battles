import { Headphones, Music2, Zap } from "lucide-react";
import BattleMenu from "../components/BattleMenu";
import { useAuth } from "../context/auth-context";

const features = [
  { Icon: Music2, title: "Choose your tracks", text: "Build a non trivial lineup to crush your opponent." },
  { Icon: Headphones, title: "Listen layer by layer", text: "Hear the song evolve as every guess reveals more." },
  { Icon: Zap, title: "Score fast", text: "Guess it early to claim the most points." },
];

export default function Lobby() {
  const { user } = useAuth();

  return (
    <div className="px-4 py-12 sm:px-6 sm:py-16 lg:py-20">
      <div className="mx-auto grid max-w-6xl items-center gap-12 lg:grid-cols-[1.05fr_0.95fr] lg:gap-20">
        <section aria-labelledby="lobby-heading">
          <p className="mb-4 text-sm font-semibold uppercase tracking-[0.22em] text-primary">Welcome back, {user?.username}</p>
          <h1 id="lobby-heading" className="max-w-2xl text-5xl font-bold leading-[1.05] tracking-[-0.04em] sm:text-6xl">
            Your music taste.<br /><span className="text-primary">Head to head.</span>
          </h1>
          <p className="mt-6 max-w-xl text-lg leading-8 text-muted">
            Challenge a friend to guess your songs from isolated tracks. Fewer clues means more points!
          </p>

          <ul className="mt-10 grid gap-4 sm:grid-cols-3 lg:grid-cols-1 xl:grid-cols-3">
            {features.map(({ Icon, title, text }) => (
              <li key={title} className="rounded-2xl border border-white/8 bg-white/[0.035] p-4">
                <Icon aria-hidden="true" className="mb-3 size-4 text-primary" />
                <h2 className="font-semibold">{title}</h2>
                <p className="mt-1 text-sm leading-6 text-muted">{text}</p>
              </li>
            ))}
          </ul>
        </section>

        <BattleMenu />
      </div>
    </div>
  );
}
