import {
  BrowserRouter,
  Navigate,
  Outlet,
  Route,
  Routes,
} from "react-router-dom";
import NavBar from "./components/Navbar";
import ProtectedRoute from "./components/ProtectedRoute";
import { AuthProvider } from "./context/AuthContext";
import Battle from "./pages/Battle";
import Lobby from "./pages/Lobby";
import Login from "./pages/Login";
import Profile from "./pages/Profile";
import Register from "./pages/Register";

function AuthenticatedLayout() {
  return (
    <div className="flex min-h-dvh flex-col bg-app text-foreground">
      <NavBar />

      <main className="min-h-0 flex-1 bg-app">
        <Outlet />
      </main>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <div className="min-h-dvh w-full bg-app">
          <Routes>
            <Route
              path="/login"
              element={<Login />}
            />

            <Route
              path="/register"
              element={<Register />}
            />

            <Route
              element={
                <ProtectedRoute>
                  <AuthenticatedLayout />
                </ProtectedRoute>
              }
            >
              <Route
                index
                element={<Lobby />}
              />

              <Route
                path="battle/:battleId"
                element={<Battle />}
              />

              <Route
                path="profile/:id"
                element={<Profile />}
              />
            </Route>

            <Route
              path="*"
              element={
                <Navigate
                  to="/"
                  replace
                />
              }
            />
          </Routes>
        </div>
      </BrowserRouter>
    </AuthProvider>
  );
}
