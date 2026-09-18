import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import Navbar from "./Navbar";

const authMock = vi.hoisted(() => ({
  logout: vi.fn(),
}));

vi.mock("@/context/auth-context", () => ({
  useAuth: () => ({
    user: { id: "user-1", username: "player" },
    logout: authMock.logout,
  }),
}));

describe("Navbar", () => {
  beforeEach(() => {
    authMock.logout.mockReset();
  });

  it("shows a useful error when logout fails", async () => {
    const user = userEvent.setup();
    authMock.logout.mockRejectedValue(new Error("Server unavailable"));
    render(<MemoryRouter><Navbar /></MemoryRouter>);

    await user.click(screen.getByRole("button", { name: "Log out" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Server unavailable");
    expect(screen.getByRole("button", { name: "Log out" })).toBeEnabled();
  });
});
