import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import PendingBattle from "./PendingBattle";

describe("PendingBattle", () => {
  it("copies the invite code and confirms success", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(<PendingBattle inviteCode="ABC123" />);

    expect(screen.getByText("ABC123")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Copy code" }));

    expect(writeText).toHaveBeenCalledWith("ABC123");
    expect(screen.getByRole("status")).toHaveTextContent(
      "Invite code copied.",
    );
  });

  it("gives manual-copy guidance when clipboard access fails", async () => {
    const user = userEvent.setup();
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: vi.fn().mockRejectedValue(new Error("Permission denied")),
      },
    });

    render(<PendingBattle inviteCode="XYZ789" />);
    await user.click(screen.getByRole("button", { name: "Copy code" }));

    expect(screen.getByRole("status")).toHaveTextContent(
      "Could not copy automatically. Select and copy the code below.",
    );
  });
});
