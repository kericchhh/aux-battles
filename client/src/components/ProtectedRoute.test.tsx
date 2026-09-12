import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
} from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuth } from "../context/auth-context";
import ProtectedRoute from "./ProtectedRoute";

vi.mock("../context/auth-context", () => ({
  useAuth: vi.fn(),
}));

const useAuthMock = vi.mocked(useAuth);

function LoginDestination() {
  const location = useLocation();
  const state = location.state as { from?: string } | null;

  return <p>Login destination: {state?.from}</p>;
}

function renderProtectedRoute(initialEntry = "/battle/123?invite=yes") {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route
          path="*"
          element={
            <ProtectedRoute>
              <p>Private content</p>
            </ProtectedRoute>
          }
        />
        <Route path="/login" element={<LoginDestination />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("ProtectedRoute", () => {
  const refresh = vi.fn();

  beforeEach(() => {
    refresh.mockReset();
    useAuthMock.mockReset();
  });

  it("shows a session check while authentication is loading", () => {
    useAuthMock.mockReturnValue({
      user: null,
      loading: true,
      error: null,
      login: vi.fn(),
      logout: vi.fn(),
      refresh,
    });

    renderProtectedRoute();

    expect(screen.getByRole("status")).toHaveTextContent(
      "Checking your session…",
    );
  });

  it("offers to retry when the session check fails", async () => {
    const user = userEvent.setup();
    useAuthMock.mockReturnValue({
      user: null,
      loading: false,
      error: new Error("Network unavailable"),
      login: vi.fn(),
      logout: vi.fn(),
      refresh,
    });

    renderProtectedRoute();

    expect(screen.getByText("Could not check your session.")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(refresh).toHaveBeenCalledOnce();
  });

  it("redirects signed-out users and preserves their intended location", () => {
    useAuthMock.mockReturnValue({
      user: null,
      loading: false,
      error: null,
      login: vi.fn(),
      logout: vi.fn(),
      refresh,
    });

    renderProtectedRoute();

    expect(
      screen.getByText("Login destination: /battle/123?invite=yes"),
    ).toBeVisible();
  });

  it("renders private content for an authenticated user", () => {
    useAuthMock.mockReturnValue({
      user: {
        id: "user-1",
        username: "player-one",
        email: "player@example.com",
      },
      loading: false,
      error: null,
      login: vi.fn(),
      logout: vi.fn(),
      refresh,
    });

    renderProtectedRoute();

    expect(screen.getByText("Private content")).toBeVisible();
  });
});
