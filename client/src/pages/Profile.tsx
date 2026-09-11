import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { errorMessage } from "../api/client";
import { getProfile } from "../api/profile";
import ErrorBanner from "../components/ErrorBanner";

export default function Profile() {
  const { id } = useParams();

  const profile = useQuery({
    queryKey: ["profile", id],
    queryFn: ({ signal }) => getProfile(id!, signal),
    enabled: Boolean(id),
  });

  if (profile.isPending) {
    return <p className="p-8 text-muted">Loading profile…</p>;
  }

  if (profile.error || !profile.data) {
    return (
      <div className="p-8">
        <ErrorBanner message={errorMessage(profile.error)} />
        <Link to="/" className="mt-4 inline-block text-primary">
          Return to lobby
        </Link>
      </div>
    );
  }

  const user = profile.data;
  const losses = user.battlesPlayed - user.wins - user.draws;

  return (
    <div className="bg-app min-h-full overflow-y-auto px-4 py-10 text-foreground">
      <section className="mx-auto max-w-3xl rounded-2xl border border-primary/50 bg-surface p-8">
        <div className="flex items-center gap-5">
          {user.avatarUrl ? (
            <img
              src={user.avatarUrl}
              alt=""
              className="size-24 rounded-full object-cover"
            />
          ) : (
            <div className="grid size-24 place-items-center rounded-full bg-primary text-3xl">
              {user.username[0]?.toUpperCase()}
            </div>
          )}

          <div>
            <h1 className="text-3xl font-bold">{user.username}</h1>
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
            <div key={label} className="rounded-xl bg-white/5 p-4 text-center">
              <dt className="text-sm text-muted">{label}</dt>
              <dd className="mt-1 text-2xl font-bold">{value}</dd>
            </div>
          ))}
        </dl>
      </section>
    </div>
  );
}
