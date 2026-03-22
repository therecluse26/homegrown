# Design System Strategy: The Curated Hearth

## 1. Overview & Creative North Star
The "Creative North Star" for this design system is **The Curated Hearth**.

In an era of cluttered ed-tech, this system rejects the "dashboard-of-widgets" aesthetic. Instead, it treats the homeschooling journey as a high-end editorial experience. We combine the warmth of a physical home library with the precision of a modern digital atelier.

To break the "template" look, we utilize **intentional asymmetry**. Hero sections for Parents might feature a large, offset `display-lg` headline that overlaps a floating `surface-container-lowest` card, while Student environments use the same depth principles to create a "pathway" feel. We prioritize breathing room over density, ensuring that "calm organization" isn't just a buzzword, but a spatial reality.

---

## 2. Colors: Tonal Depth & Soul
This palette moves away from sterile whites and harsh blacks, opting for a base of `#faf9f5` (Surface) to reduce eye strain and provide a "paper-like" warmth.

### The "No-Line" Rule
**Standard 1px solid borders are strictly prohibited for sectioning.**
Structural boundaries must be defined solely through background color shifts. To separate a sidebar from a main feed, transition from `surface` to `surface-container-low`. For a highlighted resource, use `surface-container-high`. This creates a seamless, "molded" look rather than a fragmented one.

### Surface Hierarchy & Nesting
Treat the UI as a series of physical layers.
* **Base:** `surface` (#faf9f5).
* **Secondary Content Area:** `surface-container` (#efeeea).
* **Interactive Cards:** `surface-container-lowest` (#ffffff) to create a natural "lift" against the creamier background.

### The "Glass & Gradient" Rule
To add a signature "editorial" polish:
* **Floating Navigation:** Use `surface-container-low` at 80% opacity with a `backdrop-blur` of 20px.
* **Signature Textures:** For high-impact areas (e.g., "Start Lesson" or Parent "Monthly Review"), use a subtle linear gradient from `primary` (#0c5252) to `primary-container` (#2d6a6a) at a 135-degree angle. This adds a "weighted" professional feel that flat fills cannot achieve.

---

## 3. Typography: The Editorial Voice
We use two distinct typefaces to balance authority and approachability.

* **Headlines (Plus Jakarta Sans):** Our "Display" and "Headline" scales use this font to provide a modern, geometric clarity. High-end layouts should utilize `display-lg` (3.5rem) with tighter letter-spacing (-0.02em) to create a bold, "magazine" masthead feel.
* **Body & Utility (Manrope):** Chosen for its humanist qualities and exceptional legibility. All long-form educational content and parent reports should use `body-lg` (1rem) with a generous line-height (1.6) to ensure the interface never feels "pretentious" or cramped.

**The Identity Blend:** By pairing the sophisticated, sharp terminals of Plus Jakarta Sans with the friendly, open counters of Manrope, we bridge the gap between "Professional Tool" (Parent) and "Inviting Classroom" (Student).

---

## 4. Elevation & Depth: Tonal Layering
We do not use shadows to show "importance"; we use tonal light.

* **The Layering Principle:** Depth is achieved by "stacking." A `surface-container-lowest` card placed on a `surface-container-low` background creates a soft, natural lift.
* **Ambient Shadows:** If a floating element (like a Modal or a Student "Badge" popup) requires a shadow, use a large blur (32px+) at 4% opacity using a tint of the `on-surface` color (#1b1c1a). This mimics soft, natural sunlight rather than digital "drop shadows."
* **The "Ghost Border" Fallback:** If accessibility requirements demand a border, use the `outline-variant` (#bfc8c8) at 20% opacity. Never use 100% opacity lines.
* **Glassmorphism:** For Student progress overlays, use semi-transparent `secondary-container` (#fdd6b4) with a heavy blur to allow the colorful lesson content to "glow" through the interface.

---

## 5. Components

### Buttons
* **Primary:** Solid `primary` (#0c5252) with `on-primary` (#ffffff) text. Use `md` (0.75rem) corner radius. For Parent actions, use `label-md` uppercase; for Students, use `title-sm` sentence case.
* **Secondary:** `secondary-container` fill with `on-secondary-container` text. No border.
* **Tertiary:** No fill. Use `primary` text. Interaction state: `surface-container-low` background on hover.

### Cards & Lists
* **Forbid Dividers:** Do not use lines to separate list items. Use `spacing-4` (1.4rem) of vertical whitespace or an alternating `surface` / `surface-container-low` background shift.
* **Card Styling:** Use `xl` (1.5rem) corner radius for Student content to feel "inviting" and `lg` (1rem) for Parent data to feel "sophisticated."

### Input Fields
* **Style:** Minimalist. Use `surface-container-highest` (#e3e2df) as a subtle background fill with a `none` border. On focus, transition to a `ghost border` using `primary`.
* **Feedback:** Error states use `error` (#ba1a1a) for text, but the field background should shift to `error-container` (#ffdad6) for high-glance recognition.

### Specialized Components: "The Learning Path"
* **The Progress Ribbon:** A thick, `lg` (1rem) rounded bar using `tertiary-fixed` (#ffdf96) with a `primary` fill for the active progress.
* **The Parent "Insight" Card:** A `surface-container-lowest` card with an `outline-variant` 20% "Ghost Border" and a subtle gradient header.

---

## 6. Do's and Don'ts

### Do
* **Use Asymmetry:** Offset your headline from your body text using the `12` (4rem) spacing token to create a bespoke, high-end feel.
* **Embrace Whitespace:** If a section feels "crowded," double the spacing using the `16` or `20` tokens.
* **Prioritize Hierarchy:** Use `display-sm` for Parent dashboards to convey "authority," but use `headline-lg` for Students to feel "engaging."

### Don't
* **Don't use 1px lines:** Even for tables, use background color tiers (`surface-container-low` vs `surface-container-high`) to define rows.
* **Don't use "Pure" Black:** Always use `on-surface` (#1b1c1a) for text to maintain the "warm" earth-tone aesthetic.
* **Don't over-radiate:** Avoid `full` (pill) shapes for everything; reserve them for small `Chips`. Use the `lg` and `xl` tokens for primary containers to maintain a modern, architectural structure.