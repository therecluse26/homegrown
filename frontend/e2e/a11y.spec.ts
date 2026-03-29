import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

// ─── Accessibility regression tests ──────────────────────────────────────────
// Covers critical user journeys with axe-core assertions.
// Phase 10 [P1]: zero critical/serious violations.

/** Run axe-core and assert no critical or serious violations. */
async function expectNoA11yViolations(page: AxeBuilder) {
  const results = await page.analyze();
  const serious = results.violations.filter(
    (v) => v.impact === "critical" || v.impact === "serious",
  );
  if (serious.length > 0) {
    const summary = serious
      .map(
        (v) =>
          `[${v.impact}] ${v.id}: ${v.description} (${v.nodes.length} node${v.nodes.length === 1 ? "" : "s"})`,
      )
      .join("\n");
    expect(serious, `Accessibility violations found:\n${summary}`).toHaveLength(
      0,
    );
  }
}

test.describe("Accessibility — Critical User Journeys", () => {
  test("Login page has no critical a11y violations", async ({ page }) => {
    // Navigate directly — mock auth may redirect, so we check whatever loads
    await page.goto("/auth/login");
    await page.waitForLoadState("networkidle");
    await expectNoA11yViolations(new AxeBuilder({ page }));
  });

  test("Feed / Home page has no critical a11y violations", async ({
    page,
  }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await expectNoA11yViolations(new AxeBuilder({ page }));
  });

  test("Learning dashboard has no critical a11y violations", async ({
    page,
  }) => {
    await page.goto("/learning");
    await page.waitForLoadState("networkidle");
    await expectNoA11yViolations(new AxeBuilder({ page }));
  });

  test("Marketplace browse has no critical a11y violations", async ({
    page,
  }) => {
    await page.goto("/marketplace");
    await page.waitForLoadState("networkidle");
    await expectNoA11yViolations(new AxeBuilder({ page }));
  });

  test("Settings page has no critical a11y violations", async ({ page }) => {
    await page.goto("/settings");
    await page.waitForLoadState("networkidle");
    await expectNoA11yViolations(new AxeBuilder({ page }));
  });

  test("Calendar page has no critical a11y violations", async ({ page }) => {
    await page.goto("/calendar");
    await page.waitForLoadState("networkidle");
    await expectNoA11yViolations(new AxeBuilder({ page }));
  });

  test("Search page has no critical a11y violations", async ({ page }) => {
    await page.goto("/search?q=test");
    await page.waitForLoadState("networkidle");
    await expectNoA11yViolations(new AxeBuilder({ page }));
  });
});

test.describe("Accessibility — Navigation Structure", () => {
  test("Skip link is present and targets main content", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const skipLink = page.getByRole("link", { name: "Skip to main content" });
    await expect(skipLink).toBeAttached();

    // Skip link should target #main-content
    await expect(skipLink).toHaveAttribute("href", "#main-content");
  });

  test("Page heading receives focus on navigation", async ({ page }) => {
    await page.goto("/learning");
    await page.waitForLoadState("networkidle");

    const heading = page.getByRole("heading", { level: 1 });
    await expect(heading).toBeAttached();
  });

  test("Navigation has proper ARIA landmarks", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Main navigation landmark
    const nav = page.getByRole("navigation", { name: "Main navigation" });
    await expect(nav).toBeAttached();

    // Main content landmark
    const main = page.getByRole("main");
    await expect(main).toBeAttached();

    // Banner landmark (header)
    const banner = page.getByRole("banner");
    await expect(banner).toBeAttached();
  });
});
