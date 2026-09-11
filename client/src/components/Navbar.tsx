import { Link } from "react-router-dom";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faVolumeHigh } from "@fortawesome/free-solid-svg-icons";
import { useAuth } from "../context/auth-context";

export default function NavBar() {
  const { user, logout } = useAuth();

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

      <div className="ml-auto flex items-center gap-6">
        {user ? (
          <>
            <Link
              to={`/profile/${user.id}`}
              className="text-sm text-muted transition-colors hover:text-foreground"
            >
              Profile
            </Link>

            <button
              type="button"
              onClick={() => void logout()}
              className="text-sm text-muted transition-colors hover:text-foreground"
            >
              Log out
            </button>
          </>
        ) : (
          <>
            <Link
              to="/register"
              className="text-sm text-muted transition-colors hover:text-foreground"
            >
              Register
            </Link>

            <Link
              to="/login"
              className="text-sm text-muted transition-colors hover:text-foreground"
            >
              Log in
            </Link>
          </>
        )}
      </div>
    </nav>
  );
}
