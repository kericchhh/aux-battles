import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { errorMessage } from "../api/client";
import { getProfile } from "../api/profile";
import ErrorBanner from "../components/ErrorBanner";
import LoadingState from "../components/LoadingState";
import PageShell from "../components/PageShell";
import { queryKeys } from "../lib/queryKeys";

export default function Profile() {
  const { id } = useParams();

  const profile = useQuery({
    queryKey: queryKeys.profile(id ?? ""),
    queryFn: ({ signal }) => getProfile(id!, signal),
    enabled: Boolean(id),
  });

  if (profile.isPending) {
    return <LoadingState>Loading profile…</LoadingState>;
  }

  if (profile.error || !profile.data) {
    return (
      <PageShell>
        <ErrorBanner message={errorMessage(profile.error)} />
        <Link to="/" className="mt-4 inline-block text-primary">
          Return to lobby
        </Link>
      </PageShell>
    );
  }

  const user = profile.data;
  const losses = user.battlesPlayed - user.wins - user.draws;

  return (
    <PageShell>
      <section className="mx-auto max-w-3xl rounded-xl border border-white/10 bg-surface p-6 sm:p-8">
        <div className="flex items-center gap-5">
          {user.avatarUrl ? (
            <img
              src={user.avatarUrl}
              alt=""
              className="size-24 rounded-full object-cover"
            />
          ) : (
            <div aria-hidden="true" className="grid size-20 shrink-0 place-items-center rounded-2xl bg-primary text-3xl font-bold text-[#160d1d] sm:size-24">
              {user.username[0]?.toUpperCase()}
            </div>
          )}

          <div>
            <p className="mb-1 text-xs font-semibold uppercase tracking-[0.18em] text-primary">Player profile</p>
            <h1 className="text-2xl font-bold sm:text-3xl">{user.username}</h1>
            <p className="text-muted">
              Member since {new Date(user.createdAt).toLocaleDateString()}
            </p>
          </div>
        </div>

        <dl className="mt-8 grid grid-cols-2 gap-4 sm:grid-cols-4">
          {[
            ["Battles", user.battlesPlayed],
            ["Wins", user.wins],
            ["Losses", losses],
            ["Draws", user.draws],
          ].map(([label, value]) => (
            <div key={label} className="rounded-2xl border border-white/8 bg-white/[0.035] p-4 text-center">
              <dt className="text-sm text-muted">{label}</dt>
              <dd className="mt-1 text-2xl font-bold">{value}</dd>
            </div>
          ))}
        </dl>
      </section>
    </PageShell>
  );
}
