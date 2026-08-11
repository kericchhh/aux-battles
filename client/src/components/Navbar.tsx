import { Link } from "react-router-dom";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faVolumeHigh } from "@fortawesome/free-solid-svg-icons";

export default function NavBar() {
  return (
    <nav className="flex h-16 shrink-0 items-center border-b border-white/10 bg-[#090a11] px-6">
      <div className="flex items-center gap-4">
        <FontAwesomeIcon
          icon={faVolumeHigh}
          className="text-2xl text-[#6876d8]"
        />

        <Link
          to="/"
          className="text-xl font-bold text-[#f4f0f7]"
        >
          Aux Battles
        </Link>
      </div>

      <div className="ml-auto flex items-center gap-6">
        <Link
          to="/register"
          className="text-sm text-white/70 hover:text-white"
        >
          Register
        </Link>

        <Link
          to="/login"
          className="text-sm text-white/70 hover:text-white"
        >
          Log In
        </Link>
      </div>
    </nav>
  );
}
