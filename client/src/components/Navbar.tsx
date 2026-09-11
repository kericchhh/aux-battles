import { useState } from "react";
import {
  Link,
  useNavigate,
} from "react-router-dom";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faVolumeHigh } from "@fortawesome/free-solid-svg-icons";
import { useAuth } from "../context/auth-context";

const navAction = `
  rounded-lg
  border border-transparent
  px-3 py-2
  text-sm text-muted
  transition-colors
  hover:text-primary
  disabled:cursor-not-allowed
  disabled:opacity-50
`;

export default function NavBar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [isLoggingOut, setIsLoggingOut] =
    useState(false);

  async function handleLogout() {
    if (isLoggingOut) return;

    setIsLoggingOut(true);

    try {
      await logout();

      navigate("/login", {
        replace: true,
        state: {
          loggedOut: true,
        },
      });
    } finally {
      setIsLoggingOut(false);
    }
  }

  return (
    <nav className="flex h-16 shrink-0 items-center border-b border-white/10 bg-surface px-6">
      <div className="flex items-center gap-4">
        <FontAwesomeIcon
          icon={faVolumeHigh}
          className="text-2xl text-primary"
        />

        <Link
          to="/"
          className="text-xl font-bold text-foreground"
        >
          Aux Battles
        </Link>
      </div>

      <div className="ml-auto flex items-center gap-2">
        {user ? (
          <>
            <Link
              to={`/profile/${user.id}`}
              className={navAction}
            >
              Profile
            </Link>

            <button
              type="button"
              disabled={isLoggingOut}
              onClick={() => void handleLogout()}
              className={navAction}
            >
              {isLoggingOut
                ? "Logging out…"
                : "Log out"}
            </button>
          </>
        ) : (
          <>
            <Link
              to="/register"
              className={navAction}
            >
              Register
            </Link>

            <Link
              to="/login"
              className={navAction}
            >
              Log in
            </Link>
          </>
        )}
      </div>
    </nav>
  );
}
