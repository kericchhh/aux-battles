import { createContext, useContext } from "react";
import type { User } from "../api/auth";
export const AuthContext = createContext<{
  user: User | null; loading: boolean; error: Error | null;
  login: (identifier: string, password: string) => Promise<void>;
  logout: () => Promise<void>; refresh: () => void;
} | null>(null);
export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth requires AuthProvider");
  return value;
}
