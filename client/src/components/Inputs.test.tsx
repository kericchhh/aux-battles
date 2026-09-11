import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import Input from "./Inputs";

describe("Input", () => {
  it("forwards native input properties and change events", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(
      <Input
        aria-label="Song title"
        className="w-full"
        maxLength={10}
        onChange={onChange}
      />,
    );

    const input = screen.getByRole("textbox", { name: "Song title" });
    expect(input).toHaveClass("w-full", "bg-neutral-800");
    expect(input).toHaveAttribute("maxlength", "10");

    await user.type(input, "Halo");
    expect(input).toHaveValue("Halo");
    expect(onChange).toHaveBeenCalledTimes(4);
  });
});
