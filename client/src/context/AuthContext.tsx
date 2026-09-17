import {
  useEffect,
  type ReactNode,
} from "react";
import {
  useQuery,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";
import * as auth from "@/api/auth";
import { queryKeys } from "@/lib/queryKeys";
import type { User } from "@/lib/types/auth";
import { AuthContext } from "./auth-context";

function replaceCachedUser(cache: QueryClient, user: User | null) {
  cache.removeQueries({ predicate: (query) => query.queryKey[0] !== "me" });
  cache.setQueryData(queryKeys.me, user);
}

export function AuthProvider({
  children,
}: {
  children: ReactNode;
}) {
  const cache = useQueryClient();

  const me = useQuery({
    queryKey: queryKeys.me,

    queryFn: ({ signal }) =>
      auth.getMe(signal),

    retry: false,
    refetchOnWindowFocus: true,
    refetchInterval: 60_000,
  });

  useEffect(() => {
    function expireSession() {
      void cache.cancelQueries();

      replaceCachedUser(cache, null);
    }

    window.addEventListener(
      "session:expired",
      expireSession,
    );

    return () => {
      window.removeEventListener(
        "session:expired",
        expireSession,
      );
    };
  }, [cache]);

  async function login(
    identifier: string,
    password: string,
  ) {
    const user = await auth.login({
      identifier,
      password,
    });

    await cache.cancelQueries();

    replaceCachedUser(cache, user);
  }

  async function logout() {
    await auth.logout();
    await cache.cancelQueries();

    replaceCachedUser(cache, null);
  }

  function refresh() {
    void me.refetch();
  }

  return (
    <AuthContext.Provider
      value={{
        user: me.data ?? null,
        loading: me.isPending,
        error: me.error,
        login,
        logout,
        refresh,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
