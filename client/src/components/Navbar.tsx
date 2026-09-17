import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { AudioLines, House, LogOut, UserRound } from "lucide-react";
import { useAuth } from "@/context/auth-context";

const navAction = "inline-flex min-h-10 items-center gap-2 rounded-xl border border-transparent px-3 py-2 text-sm text-muted transition-colors hover:bg-white/5 hover:text-foreground focus-visible:outline-2 focus-visible:outline-primary disabled:cursor-not-allowed disabled:opacity-50";

export default function NavBar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  async function handleLogout() {
    if (isLoggingOut) return;
    setIsLoggingOut(true);
    try {
      await logout();
      navigate("/login", { replace: true, state: { loggedOut: true } });
    } finally {
      setIsLoggingOut(false);
    }
  }

  return (
    <nav aria-label="Main navigation" className="sticky top-0 z-40 flex h-16 shrink-0 items-center border-b border-white/8 bg-canvas px-4 sm:px-6">
      <Link to="/" aria-label="Aux Battles home" className="group flex items-center gap-3 rounded-xl">
        <span className="grid size-8 place-items-center rounded-lg border border-white/10 bg-surface-raised text-primary transition-colors group-hover:border-white/20">
          <AudioLines aria-hidden="true" className="size-4" />
        </span>
        <span className="text-lg font-bold tracking-tight text-foreground sm:text-xl">AUX <span className="text-primary">BATTLES</span></span>
      </Link>

      <div className="ml-auto flex items-center gap-1 sm:gap-2">
        {user ? (
          <>
            <Link aria-label="Lobby" title="Lobby" to="/" className={navAction}>
              <House aria-hidden="true" className="size-4" /><span className="hidden sm:inline">Lobby</span>
            </Link>
            <Link aria-label="Profile" title="Profile" to={`/profile/${user.id}`} className={navAction}>
              <UserRound aria-hidden="true" className="size-4" /><span className="hidden sm:inline">Profile</span>
            </Link>
            <button aria-label="Log out" title="Log out" type="button" disabled={isLoggingOut} onClick={() => void handleLogout()} className={navAction}>
              <LogOut aria-hidden="true" className="size-4" /><span className="hidden sm:inline">{isLoggingOut ? "Logging out…" : "Log out"}</span>
            </button>
          </>
        ) : (
          <><Link to="/register" className={navAction}>Register</Link><Link to="/login" className={navAction}>Log in</Link></>
        )}
      </div>
    </nav>
  );
}
