import { Outlet } from "react-router-dom";
import Navbar from "./Navbar";

export default function AppLayout() {
  return (
    <div className="flex min-h-dvh flex-col bg-app text-foreground">
      <a href="#main-content" className="fixed left-4 top-3 z-50 -translate-y-20 rounded-lg bg-primary px-4 py-2 font-semibold text-black transition-transform focus:translate-y-0">
        Skip to content
      </a>
      <Navbar />

      <main id="main-content" className="min-h-0 flex-1">
        <Outlet />
      </main>
    </div>
  );
}
