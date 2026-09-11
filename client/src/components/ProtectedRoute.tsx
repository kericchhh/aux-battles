import type { ReactNode } from "react";
import {
  Navigate,
  useLocation,
} from "react-router-dom";
import { useAuth } from "../context/auth-context";
import Button from "./Button";
import LoadingState from "./LoadingState";

export default function ProtectedRoute({
  children,
}: {
  children: ReactNode;
}) {
  const {
    user,
    loading,
    error,
    refresh,
  } = useAuth();

  const location = useLocation();

  if (loading) {
    return (
      <LoadingState>
        Checking your session…
      </LoadingState>
    );
  }

  if (error) {
    return (
      <div className="space-y-4 p-8 text-foreground">
        <p>
          Could not check your session.
        </p>

        <Button
          variant="secondary"
          onClick={refresh}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (!user) {
    return (
      <Navigate
        to="/login"
        replace
        state={{
          from:
            location.pathname +
            location.search,
        }}
      />
    );
  }

  return children;
}
