import { useEffect, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import * as auth from "../api/auth";
import { AuthContext } from "./auth-context";
export function AuthProvider({children}: {children: ReactNode}) {
  const cache = useQueryClient();
  const me = useQuery({queryKey: ["me"], queryFn: ({signal}) => auth.getMe(signal), retry: false,
    refetchOnWindowFocus: true, refetchInterval: 60000});
  useEffect(() => {
    const expire = () => { void cache.cancelQueries(); cache.removeQueries({predicate: q => q.queryKey[0] !== "me"}); cache.setQueryData(["me"], null); };
    window.addEventListener("session:expired", expire);
    return () => window.removeEventListener("session:expired", expire);
  }, [cache]);
  async function login(identifier: string, password: string) {
    await cache.cancelQueries({queryKey: ["me"]});
    cache.removeQueries({ queryKey: ["me"]})
    const user = await auth.login({identifier, password});
    cache.setQueryData(["me"], user)
    await cache.cancelQueries(); cache.clear(); cache.setQueryData(["me"], user);
  }
  async function logout() {
    await auth.logout(); await cache.cancelQueries(); cache.clear(); cache.setQueryData(["me"], null);
  }
  return <AuthContext.Provider value={{user: me.data ?? null, loading: me.isPending, error: me.error,
    login, logout, refresh: () => { void me.refetch(); }}}>{children}</AuthContext.Provider>;
}
