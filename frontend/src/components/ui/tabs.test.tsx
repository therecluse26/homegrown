import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect } from "vitest";
import { Tabs } from "./tabs";

const tabs = [
  { id: "alpha", label: "Alpha", content: <div>Alpha content</div> },
  { id: "beta", label: "Beta", content: <div>Beta content</div> },
  { id: "gamma", label: "Gamma", content: <div>Gamma content</div> },
];

describe("Tabs", () => {
  it("renders all tab buttons", () => {
    render(<Tabs tabs={tabs} />);
    expect(screen.getAllByRole("tab")).toHaveLength(3);
    expect(screen.getByRole("tab", { name: "Alpha" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Beta" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Gamma" })).toBeInTheDocument();
  });

  it("shows first tab content by default", () => {
    render(<Tabs tabs={tabs} />);
    expect(screen.getByText("Alpha content")).toBeInTheDocument();
    expect(screen.queryByText("Beta content")).not.toBeInTheDocument();
  });

  it("respects defaultTab prop", () => {
    render(<Tabs tabs={tabs} defaultTab="beta" />);
    expect(screen.queryByText("Alpha content")).not.toBeInTheDocument();
    expect(screen.getByText("Beta content")).toBeInTheDocument();
  });

  it("switches content on tab click", async () => {
    const user = userEvent.setup();
    render(<Tabs tabs={tabs} />);

    await user.click(screen.getByRole("tab", { name: "Beta" }));
    expect(screen.queryByText("Alpha content")).not.toBeInTheDocument();
    expect(screen.getByText("Beta content")).toBeInTheDocument();
  });

  it("sets aria-selected on the active tab", async () => {
    const user = userEvent.setup();
    render(<Tabs tabs={tabs} />);

    expect(screen.getByRole("tab", { name: "Alpha" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByRole("tab", { name: "Beta" })).toHaveAttribute(
      "aria-selected",
      "false",
    );

    await user.click(screen.getByRole("tab", { name: "Beta" }));
    expect(screen.getByRole("tab", { name: "Alpha" })).toHaveAttribute(
      "aria-selected",
      "false",
    );
    expect(screen.getByRole("tab", { name: "Beta" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("renders a tabpanel with correct aria attributes", () => {
    render(<Tabs tabs={tabs} defaultTab="alpha" />);
    const panel = screen.getByRole("tabpanel");
    expect(panel).toHaveAttribute("id", "panel-alpha");
    expect(panel).toHaveAttribute("aria-labelledby", "tab-alpha");
  });

  it("navigates with ArrowRight key", async () => {
    const user = userEvent.setup();
    render(<Tabs tabs={tabs} />);

    // Focus the active tab
    screen.getByRole("tab", { name: "Alpha" }).focus();
    await user.keyboard("{ArrowRight}");

    expect(screen.getByText("Beta content")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Beta" })).toHaveFocus();
  });

  it("navigates with ArrowLeft key and wraps around", async () => {
    const user = userEvent.setup();
    render(<Tabs tabs={tabs} />);

    screen.getByRole("tab", { name: "Alpha" }).focus();
    await user.keyboard("{ArrowLeft}");

    // Wraps to last tab
    expect(screen.getByText("Gamma content")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Gamma" })).toHaveFocus();
  });

  it("navigates to first tab with Home key", async () => {
    const user = userEvent.setup();
    render(<Tabs tabs={tabs} defaultTab="gamma" />);

    screen.getByRole("tab", { name: "Gamma" }).focus();
    await user.keyboard("{Home}");

    expect(screen.getByText("Alpha content")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Alpha" })).toHaveFocus();
  });

  it("navigates to last tab with End key", async () => {
    const user = userEvent.setup();
    render(<Tabs tabs={tabs} />);

    screen.getByRole("tab", { name: "Alpha" }).focus();
    await user.keyboard("{End}");

    expect(screen.getByText("Gamma content")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Gamma" })).toHaveFocus();
  });

  it("sets tabIndex=0 only on active tab", () => {
    render(<Tabs tabs={tabs} />);
    expect(screen.getByRole("tab", { name: "Alpha" })).toHaveAttribute(
      "tabIndex",
      "0",
    );
    expect(screen.getByRole("tab", { name: "Beta" })).toHaveAttribute(
      "tabIndex",
      "-1",
    );
    expect(screen.getByRole("tab", { name: "Gamma" })).toHaveAttribute(
      "tabIndex",
      "-1",
    );
  });
});
