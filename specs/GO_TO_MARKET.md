# Homegrown Academy — Go-to-Market Plan

This document captures the strategy for bridging the cold-start gap: getting from zero
to thousands of active families before network effects take hold.

---

## Core Thesis

Homegrown Academy launches as a **standalone homeschool management tool** and progressively
enables social and marketplace features as user density grows. Each phase delivers
complete, self-contained value — no phase feels like an empty platform waiting for users.

> "Come for the tool, stay for the network."

---

## 1. Methodology Beachhead

**Launch to one methodology community first, not "homeschoolers" broadly.**

Recommended first target: a single, tight-knit community (Charlotte Mason, Classical, or
Unschooling — to be validated). These communities are tribal, have established gathering
places (forums, Facebook groups, Instagram hashtags, podcasts), and word-of-mouth travels
fast within them.

### Why this works with our product

- Methodology-as-config means the entire UX speaks a community's language from day one —
  terminology, tool activation, learning workflows are all native to their pedagogy.
- A social layer where every member shares the same educational worldview creates
  immediate belonging (once enabled in Phase B).
- Marketplace creators can target a known audience with methodology-specific content.

### Execution

1. Identify the target methodology community through research and founder network.
2. Engage 5–10 families from that community as design partners pre-launch.
3. Validate the onboarding flow, terminology, and tool configuration for that methodology.
4. Launch publicly within that community's channels.
5. Only expand to the next methodology after reaching critical mass in the first.

---

## 2. Progressive Feature Rollout

Features are gated using the existing feature flag system (`admin_feature_flags` table)
and rolled out in phases as user density justifies them.

### Phase A: The Solo Tool (Zero Network Required)

**Target: Launch → first 50 families**

Enabled features:
- Onboarding (methodology selection, family profile, child setup)
- Learning tools (activities, journals, reading lists, quizzes, sequences, videos,
  progress tracking, student sessions)
- Compliance reporting (attendance, portfolios, transcripts)
- Calendar and planning
- Settings, privacy controls, data export
- Notification system (in-app + email)

Disabled features:
- Social (feed, friends, messaging, groups, events)
- Marketplace (browse, purchase, creator tools)
- Recommendations engine
- Search (limited content to search at this stage)

**Value proposition:** A complete homeschool management and compliance platform that works
for one family in isolation. Every user gets full value on day one.

### Phase B: The Cohort (Micro-Network)

**Target: ~50–100 families**

Enable:
- Groups (start with admin-curated methodology-specific groups)
- Events (virtual co-op meetups, study groups)
- Family profiles (opt-in visibility)

Hold back:
- Feed, friends, direct messaging (require more density)
- Marketplace (require creator supply)

**Value proposition:** "There are other families like you here." Controlled introduction
of community features in curated spaces, so nothing feels empty.

### Phase C: The Marketplace (Supply Meets Demand)

**Target: ~200–500 families, 10–20 active creators**

Enable:
- Marketplace browse and purchasing
- Creator onboarding and dashboard
- Recommendations engine

**Value proposition:** Methodology-specific curriculum and resources, sold by creators
who understand your approach. Revenue share incentivises creator investment.

### Phase D: Full Social

**Target: ~500+ families**

Enable:
- Feed and timeline
- Friends and direct messaging
- Discovery and search
- Full social features

**Value proposition:** A living community of like-minded families. By this point, enough
activity exists that the social layer feels alive, not empty.

---

## 3. Creator Seeding (Before Phase C)

The marketplace is a two-sided market with its own cold-start problem. Solve the supply
side first.

### Execution

1. Recruit 10–20 curriculum creators in the target methodology **before** opening the
   marketplace to consumers.
2. Provide early access to creator dashboard, quiz builder, and sequence builder.
3. Offer launch-period incentives (higher revenue share, featured placement, direct
   support).
4. Creators bring their existing audiences when marketplace opens — "My new curriculum
   is on Homegrown Academy."
5. Families see a populated catalog on day one, not an empty shelf.

---

## 4. Compliance as a Trojan Horse

### The insight

In high-regulation states (NY, PA, OH, VA, and others), families are **legally required**
to track attendance, submit assessments, and maintain portfolios. Most cobble this
together with spreadsheets, binders, and prayer. This is a pain point with no network
dependency — pure standalone value.

### Execution

1. Nail the compliance UX for 3–5 high-regulation states first.
2. Market directly to families in those states: "Meet your state's requirements in
   minutes, not hours."
3. Families adopt for the legal obligation and discover the learning tools, then stay.
4. Compliance frontend (currently placeholder) is a **high-priority build** for launch
   readiness.

---

## 5. Acquisition Channels

### 5a. Content-Led SEO

Homeschooling parents are voracious researchers. They spend months planning before each
school year. Create high-value content:

- **Methodology comparison guides** — the platform's multi-methodology architecture is
  uniquely qualified to produce these.
- **"How to start homeschooling in [state]"** — legal requirements vary by state, and
  this is a top search query.
- **Planning templates and checklists** — free resources that demonstrate what the
  platform automates.

Funnel: free resource → email list → product launch announcement.

### 5b. Micro-Influencer Partnerships

The homeschooling world runs on trust. Target:
- Homeschool bloggers and podcasters with 2K–10K engaged followers
- Co-op leaders who organize local groups
- Curriculum reviewers on YouTube

Give them early access, listen to their feedback, let them shape the product. They become
evangelists because they feel ownership — not because they were paid.

### 5c. Homeschool Convention Circuit

Conventions (Great Homeschool Conventions, CHEA, state-level conferences) happen
March–July, exactly when families make purchasing decisions. With the product's current
completeness, a live demo is compelling:

- Walk through onboarding → methodology selection → first learning session
- Show compliance tracking for the attendee's state
- QR code to free trial signup
- Convention-goers are the highest-intent audience available.

### 5d. Co-op Seeding

Homeschool co-ops (groups of 5–30 families meeting regularly) are the atomic unit of
network effect. If one co-op adopts, you instantly have a micro-network.

- Offer a free group plan for co-ops of 5+ families.
- Build or highlight features for co-op coordination (group events, shared resources).
- One happy co-op leader talks to other co-op leaders.

Manual and slow, but each co-op is a self-sustaining cluster.

---

## 6. Free Tier Strategy

The billing system already supports tier gating via `TierGate`.

### Free tier includes (enough to be genuinely useful):
- Methodology configuration and onboarding
- Basic learning logging (activities, journals, reading lists)
- Compliance tracking (attendance, portfolio basics)
- Calendar and planning
- Data export

### Paid tier unlocks:
- Full progress analytics and reporting
- Marketplace access
- Advanced compliance features (transcripts, PDF exports)
- Full social features (when enabled)
- Priority support

The free tier must be a real product, not a crippled demo. Families should be able to run
their homeschool on the free tier. The paid tier adds depth, community, and convenience.

---

## 7. Trust Signals

### Privacy as a wedge

Privacy-first positioning is a genuine differentiator. Many homeschooling families are
privacy-conscious — it's often part of why they homeschool. Lean into this:

- Family-scoped data isolation (RLS) is structural, not policy
- COPPA compliance is by construction, not bolted on
- No GPS tracking, no PII logging, no student credentials
- CSAM detection and NCMEC reporting pipeline (responsible platform)

### Data portability

GDPR-compliant data export and one-click account deletion are already built. Market this
aggressively: **"Your data is yours. Export it anytime. Delete everything with one click."**
Most ed-tech competitors cannot make this claim.

---

## 8. Seasonal Timing

There is a massive seasonal spike every July–August when families plan their school year.

- **March–June:** Content marketing, convention presence, creator recruitment
- **July–August:** Launch push — free trial campaigns, "plan your year" messaging
- **September:** Families are locked in for the year; retention focus
- **January:** Mid-year planning adjustments; secondary acquisition window

A launch timed to the July–August planning window maximises first-year adoption.

---

## 9. Student Session as Demo Hook

The student session system (launcher, dashboard, quiz player, video player, reader,
sequence progression, parent scoring) is a complete supervised learning experience and
the most demo-able feature.

Possible lead-gen tactic:
- "Try a 30-minute learning session with your child — free, no signup required."
- Seed with sample quizzes, reading passages, and sequences for the target methodology.
- Parent sees scoring flow, progress tracking, and activity log afterward.
- Converting from "that worked great" to "I want this for our whole year" is a short leap.

---

## 10. Technical Prerequisites for Launch

### Must-build (blocks go-to-market)

| Item | Current state | Priority |
|------|--------------|----------|
| Compliance frontend (attendance, portfolios, transcripts) | Backend complete, frontend placeholder | **P0** |
| Feature flag integration layer (consumer endpoint, frontend hook, route gating) | Backend flags built, no wiring | **P0** |
| Navigation hiding for disabled features | Not implemented | **P0** |

### Should-build (strengthens launch)

| Item | Current state | Priority |
|------|--------------|----------|
| Marketplace browse + listing detail UI | Placeholder | **P1** |
| Creator dashboard + content builders | Placeholder | **P1** |
| Calendar/planning UI | Placeholder | **P1** |
| Admin feature flags UI | Placeholder | **P1** |
| Search UI | Placeholder | **P2** |

### Already done (launch-ready)

- Authentication and onboarding flows
- Learning tools (activities, journals, reading, quizzes, sequences, videos, progress)
- Student session system
- Settings, privacy controls, data export, account deletion
- Notification system
- Social layer (ready to enable when density supports it)
- Billing and subscription tier gating
- Safety and moderation pipeline

---

## 11. Success Metrics by Phase

| Phase | Families | Key metric | Signal to advance |
|-------|----------|-----------|-------------------|
| **A: Solo Tool** | 0 → 50 | Weekly active families | 60%+ WAU/MAU ratio |
| **B: Cohort** | 50 → 200 | Group participation rate | 30%+ families join a group |
| **C: Marketplace** | 200 → 500 | Creator catalog + purchases | 10+ listings, first purchases |
| **D: Full Social** | 500+ | Social engagement | Posts, messages, friend connections |

---

## 12. Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Building for wrong methodology first | Validate with 5–10 design-partner families before full launch |
| Marketplace has no supply | Seed creators before opening marketplace; curate initial catalog |
| Compliance requirements vary too much by state | Start with 3–5 states; use methodology-as-config pattern for state config |
| Feature flags don't gate cleanly | Build integration layer (P0) before launch; test progressive rollout internally |
| Free tier is too generous, no conversion | Track feature usage analytics; gate features that correlate with retention |
| Convention presence is expensive | Start with 1–2 regional conventions; measure conversion before scaling |
