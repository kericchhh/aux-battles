import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "../context/auth-context";

export default function ProtectedRoute({
  children,
}: {
  children: ReactNode;
}) {
  const { user, loading, error, refresh } = useAuth();

  if (loading) {
    return <p className="p-8 text-white">Checking your session…</p>;
  }

  if (error) {
    return (
      <div className="p-8 text-white">
        <p>Could not check your session.</p>
        <button onClick={refresh}>Retry</button>
      </div>
    );
  }

  if (!user) return <Navigate to="/login" replace />;

  return children;
}
