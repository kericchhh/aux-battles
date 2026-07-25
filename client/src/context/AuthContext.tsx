import { createContext, useContext, useState, type ReactNode } from "react";

interface AuthState {
  token: string | null;
  userId: string | null;
  username: string | null;
}

interface AuthContextValue extends AuthState {
  login: (token: string, userId: string, username: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(() => ({
    token: localStorage.getItem("authToken"),
    userId: localStorage.getItem("userId"),
    username: localStorage.getItem("username"),
  }));

  function login(token: string, userId: string, username: string) {
    localStorage.setItem("authToken", token);
    localStorage.setItem("userId", userId);
    localStorage.setItem("username", username);
    setState({ token, userId, username });
  }

  function logout() {
    localStorage.removeItem("authToken");
    localStorage.removeItem("userId");
    localStorage.removeItem("username");
    setState({ token: null, userId: null, username: null });
  }

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
