# Homegrown Academy — Project Specification

## 1. Introduction & Scope

### 1.1 Purpose

This document translates the Homegrown Academy product vision (`specs/VISION.md`) into concrete, implementable specifications. It is the authoritative reference for **what the platform must do** and the constraints it must satisfy, while remaining technology-agnostic. It does not prescribe *how* to build the system — that is the responsibility of architecture and engineering design documents that follow.

### 1.2 Relationship to VISION.md

Every requirement in this specification traces back to the vision document using `[V§n]` references (e.g., `[V§5]` refers to Vision §5 — *Core Architecture: Methodology as a First-Class Concept*). If a conflict exists between this document and the vision, the vision takes precedence on intent and the specification takes precedence on implementable detail.

### 1.3 Intended Audience

- Product owners and designers
- Engineering and architecture teams
- QA and test engineers
- Trust & Safety, Legal, and Compliance stakeholders

### 1.4 Conventions

This document uses RFC 2119 keywords:

| Keyword | Meaning |
|---------|---------|
| **MUST** / **MUST NOT** | Absolute requirement or prohibition |
| **SHOULD** / **SHOULD NOT** | Recommended unless a compelling reason exists to deviate |
| **MAY** | Truly optional; implementation is at the team's discretion |

### 1.5 Explicit Out-of-Scope

The following are **not** covered by this specification and are deferred to future documents or phases:

- Native mobile applications (iOS/Android) `[V§11]`
- Group video conferencing for virtual co-op sessions `[V§11]`
- Tutoring marketplace `[V§10]`
- Third-party integrations beyond payment processing (e.g., library APIs, standardized testing providers)
- Internationalization and localization beyond architectural readiness
- AI tutoring (data collection requirements are in-scope; the tutoring feature itself is not)

---

## 2. System Domains & Boundaries

### 2.1 Domain Decomposition

The platform is organized into the following logical domains. Each domain encapsulates a cohesive set of responsibilities and maps to one or more vision sections.

| # | Domain | Responsibilities | Vision Sections |
|---|--------|-----------------|-----------------|
| 1 | **Identity & Access** | Authentication, authorization, account lifecycle, session management | V§4, V§7 |
| 2 | **Methodology** | Methodology definitions, tool registries, philosophy modules, configuration propagation | V§5 |
| 3 | **Discovery** | Public-facing content: methodology quiz, methodology explorer, state legal guides, Homeschooling 101, advocacy content | V§6 |
| 4 | **Onboarding** | Account creation flow, family profile setup, methodology selection wizard, getting-started roadmaps, starter recommendations, community connections | V§6 |
| 5 | **Social** | Profiles, timeline/feed, comments, friends, direct messaging, groups, events, location-based discovery | V§7 |
| 6 | **Learning** | Methodology-scoped learning tools, activity logging, progress tracking, parent education content | V§8 |
| 7 | **Marketplace** | Creator onboarding, content listings, discovery, purchase flow, ratings/reviews, revenue share, payouts | V§9 |
| 8 | **AI & Recommendations** | Content suggestions, curriculum recommendations, methodology-constrained recommendation engine | V§8 |
| 9 | **Compliance & Reporting** | State-specific compliance configuration, attendance logs, assessment records, portfolios, transcripts | V§8 |
| 10 | **Trust & Safety** | CSAM detection, content moderation, user reporting, bot prevention, child safety | V§7 |
| 11 | **Billing & Subscriptions** | Free/premium tier management, marketplace transactions, creator payouts, tax compliance | V§10 |
| 12 | **Notifications** | In-app notifications, email delivery, user preferences, digest management | V§7, V§8, V§9 |
| 13 | **Search** | Full-text search across social, learning, and marketplace content; autocomplete; faceted filtering | V§7, V§8, V§9 |
| 14 | **Content & Media** | File uploads, image processing, media storage, content delivery | All |

### 2.2 Domain Boundary Rationale

Each domain SHOULD be independently deployable and maintainable. Domains communicate through well-defined contracts (see §18). The Methodology domain is **cross-cutting** — it provides configuration consumed by nearly every other domain. Content & Media is a **shared infrastructure** domain providing storage and delivery services to all other domains.

### 2.3 Cross-Cutting Concerns

The following concerns span all domains and MUST be addressed consistently:

- **Authentication & Authorization** — Every request MUST be authenticated (except public Discovery content) and authorized against the permission model (§3).
- **Audit Logging** — Security-sensitive operations MUST produce audit records.
- **Rate Limiting** — All public-facing endpoints MUST enforce rate limits.
- **Data Privacy** — All domains MUST comply with the privacy model (§17.2).
- **Methodology Configuration** — Domains that vary behavior by methodology MUST consume configuration from the Methodology domain, not embed methodology logic.

---

## 3. User Roles, Accounts & Permissions

### 3.1 Account Types

#### 3.1.1 Family Account `[V§4]`

The Family Account is the top-level entity. All platform activity is organized under a family.

- Every family MUST have at least one Parent User.
- A family MUST have a designated **primary parent** who holds billing responsibility and cannot be removed without transferring the role.
- A family MAY have one or more additional Parent Users (co-parents, guardians).
- A family MAY have zero or more Student Profiles.

#### 3.1.2 Parent User `[V§4]`

A Parent User is an authenticated individual associated with a Family Account.

- A Parent User MUST authenticate with their own credentials (email/password or OAuth).
- All Parent Users within a family MUST have equal access to all family data, including all Student Profiles.
- A Parent User MAY participate in the social layer (posts, messaging, groups, events).
- A Parent User MAY purchase marketplace content and manage subscriptions.
- A Parent User MUST be able to act on behalf of any Student Profile in the family.

#### 3.1.3 Student Profile `[V§4]`

A Student Profile represents a child within a family. Students do NOT have independent accounts. `[V§7]`

- A Student Profile MUST be created and managed by a Parent User.
- A Student Profile MUST NOT have independent login credentials in MVP.
- All Student Profile activity MUST be visible to every Parent User in the family.
- A Student Profile MUST be associated with an age or birth year, grade level (optional), and name.
- A Student Profile MAY have methodology overrides that differ from the family's primary/secondary selection (see §4.6).

#### 3.1.4 Creator Account `[V§9]`

A Creator Account enables an individual or organization to sell content through the marketplace.

- A Creator Account MAY be linked to an existing Parent User account (i.e., a homeschool parent who also creates content) or MAY stand alone.
- A Creator Account MUST complete identity verification and tax information collection before receiving payouts (see §9.1).
- A Creator Account MUST agree to marketplace terms of service, including content policies.

#### 3.1.5 Platform Administrator

- Platform Administrators are internal users with elevated access for moderation, support, and system management.
- Administrator actions MUST be audit-logged.
- Administrator access MUST follow the principle of least privilege with role-based sub-permissions (e.g., moderation-only, billing-support-only).

### 3.2 Permission Matrix

| Capability | Unauthenticated | Free Parent | Premium Parent | Student (via Parent) | Creator | Admin |
|-----------|:-:|:-:|:-:|:-:|:-:|:-:|
| View public discovery content | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| Take methodology quiz | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| Create account | ✓ | — | — | — | — | — |
| Manage family profile | — | ✓ | ✓ | — | — | ✓* |
| Select methodologies | — | ✓ | ✓ | — | — | — |
| Use basic learning tools | — | ✓ | ✓ | ✓ | — | — |
| Use advanced learning tools | — | — | ✓ | ✓** | — | — |
| View basic progress tracking | — | ✓ | ✓ | — | — | — |
| View advanced analytics | — | — | ✓ | — | — | — |
| Access compliance reporting | — | — | ✓ | — | — | — |
| Access AI recommendations | — | — | ✓ | — | — | — |
| Post to timeline | — | ✓ | ✓ | — | — | — |
| Comment on posts | — | ✓ | ✓ | — | — | — |
| Send direct messages | — | ✓ | ✓ | — | — | — |
| Join/create groups | — | ✓ | ✓ | — | — | — |
| Create/discover events | — | ✓ | ✓ | — | — | — |
| Browse marketplace | — | ✓ | ✓ | — | ✓ | ✓ |
| Purchase marketplace content | — | ✓ | ✓ | — | — | — |
| List/manage marketplace content | — | — | — | — | ✓ | ✓* |
| Receive payouts | — | — | — | — | ✓ | — |
| Moderate content | — | — | — | — | — | ✓ |
| Manage users/accounts | — | — | — | — | — | ✓ |

*Admin access is for support/moderation purposes with audit logging.
**Student access to advanced tools requires parent's family to have premium subscription.

### 3.3 Parent-Mediated Access Model `[V§7]`

- All student interactions with the platform MUST be mediated through a parent account.
- Parents MUST have complete visibility into all activity associated with their Student Profiles.
- Parents MUST approve any social connections involving their children.
- The platform MUST NOT allow direct, unmediated communication between students and non-family adults.

### 3.4 Family Account Lifecycle

| Event | Requirements |
|-------|-------------|
| **Creation** | Requires one parent with verified email. Family account is created simultaneously with the first parent user. |
| **Add co-parent** | Primary parent invites via email. Invited user creates credentials and joins the family. |
| **Add student** | Any parent in the family creates a Student Profile. |
| **Remove co-parent** | Primary parent MAY remove a co-parent. The co-parent's personal content (posts, messages) SHOULD be preserved but disassociated from the family. |
| **Transfer primary role** | Primary parent MAY transfer the primary role to another parent in the family. |
| **Remove student** | Any parent MAY remove a Student Profile. Learning data MUST be exportable before deletion. Deletion MUST be permanent after a grace period. |
| **Delete family** | Primary parent only. MUST trigger data export opportunity, followed by complete data deletion per retention policies (§16.3). Active subscriptions MUST be cancelled. Marketplace purchase history MUST be retained for financial/legal compliance. |

### 3.5 Multi-Child Families

- The platform MUST support families with multiple children of varying ages and grade levels.
- Each Student Profile MUST maintain independent learning data, progress records, and (optionally) methodology overrides.
- Learning tools MUST support selecting which student(s) an activity applies to.
- Billing MUST be family-level, not per-student. `[V§10]`
- Progress views MUST support both per-student and family-wide perspectives.

---

## 4. Methodology System (Cross-Cutting Architecture) `[V§5]`

### 4.1 Methodology as a Configuration Entity

A Methodology is a first-class platform entity — not a tag or label. Each methodology MUST be defined as a configuration object comprising:

| Component | Description |
|-----------|-------------|
| **Identity** | Unique identifier, display name, short description, icon/branding |
| **Philosophy Module** | Educational content: history, principles, practical application, "what a typical day looks like," pros/cons, recommended starting resources `[V§6]` |
| **Tool Registry** | The set of learning tools activated for this methodology (see §4.2) |
| **Content Tags** | Marketplace content tags associated with this methodology `[V§9]` |
| **Onboarding Config** | Methodology-specific onboarding steps, getting-started roadmap, and starter curriculum recommendations `[V§6]` |
| **Mastery Paths** | Beginner → intermediate → advanced parent education progression `[V§6]` |
| **Community Config** | Default group associations, mentor matching criteria `[V§6, V§7]` |

### 4.2 Tool Registry Pattern

The learning tool system follows a two-level registry:

1. **Master Tool Catalog** — The complete set of learning tools available on the platform (see §8 for the full catalog). Each tool has a unique identifier, display name, description, and configuration schema.

2. **Per-Methodology Activation** — Each methodology's Tool Registry specifies which tools from the master catalog are activated, along with any methodology-specific configuration (e.g., labels, defaults, guidance text).

3. **Family's Active Tool Set** — When a family selects one or more methodologies, the union of all activated tools across selected methodologies becomes the family's available tool set.

**Example:**
- Charlotte Mason activates: Activities, Reading Lists, Journaling & Narration, Nature Journals, Progress Tracking
- Traditional activates: Activities, Tests & Grades, Progress Tracking
- A family selecting both Charlotte Mason + Traditional sees the union: Activities, Reading Lists, Journaling & Narration, Nature Journals, Tests & Grades, Progress Tracking

### 4.3 Multi-Methodology (Eclectic) Support `[V§5]`

- Families MUST be able to select one **primary** methodology and zero or more **secondary** methodologies.
- The primary methodology SHOULD shape the default dashboard layout, terminology, and UX feel.
- There MUST NOT be an artificial "Eclectic" methodology category — eclecticism is expressed by selecting multiple methodologies.
- Families MUST be able to change their methodology selections at any time.
- Changing methodology selections MUST NOT delete historical learning data — tools that are no longer active SHOULD remain accessible in a read-only archived state.

### 4.4 Per-Domain Methodology Impact

| Domain | How Methodology Shapes It |
|--------|--------------------------|
| **Learning** | Determines which tools are available, tool labels and guidance text, suggested workflows |
| **Marketplace** | Filters and sorts content recommendations; powers methodology-scoped browsing |
| **Social** | Drives group suggestions, mentor matching, and community connections |
| **Onboarding** | Determines the getting-started roadmap, starter recommendations, and wizard content |
| **Discovery** | Powers quiz results and methodology explorer content |
| **AI & Recommendations** | Constrains recommendation engine outputs to methodology-relevant content |
| **Compliance** | MAY influence default report templates (e.g., portfolio-based vs. transcript-based) |
| **Notifications** | MAY customize notification content and terminology |

### 4.5 Methodology Data Governance

- Methodology definitions MUST be platform-managed, not user-editable.
- Adding or modifying a methodology MUST be achievable through configuration changes, not code changes.
- The initial methodology set MUST include: **Charlotte Mason, Traditional, Classical, Waldorf, Montessori, Unschooling**. `[V§5]`
- Additional methodologies (e.g., Reggio Emilia, project-based learning, eclectic-as-named) MAY be added over time through configuration.
- All methodologies MUST receive equal platform investment and support — no methodology is treated as default or preferred. `[V§12]`

### 4.6 Per-Student Methodology Overrides

- A Parent User MAY assign methodology overrides to individual Student Profiles.
- When a student has a methodology override, that student's tool set SHOULD be derived from their personal methodology selection rather than the family-level selection.
- Per-student overrides MUST NOT affect other students in the family.
- Per-student overrides are OPTIONAL — by default, students inherit the family's methodology configuration.

---

## 5. Discovery & Public Content `[V§6]`

### 5.1 Methodology Quiz Engine

#### 5.1.1 Functional Requirements

- The quiz MUST be accessible without an account. `[V§6]`
- The quiz MUST ask about family values, learning preferences, practical constraints, and child temperament.
- The quiz MUST produce ranked methodology recommendations with explanations of why each methodology fits.
- The quiz MUST NOT be a lead-capture form — it MUST provide genuine, complete results without requiring account creation. `[V§6]`
- The quiz SHOULD be shareable via URL (results page with unique ID).
- The quiz SHOULD support retaking with different answers.

#### 5.1.2 Data Model

- **Quiz Definition**: A versioned set of questions, answer options, and scoring weights. The platform MUST support updating the quiz over time without invalidating prior results.
- **Quiz Result**: Methodology scores, ranked recommendations, and explanations. Results MUST be stored with a unique, URL-safe identifier.
- **Anonymous Results**: Quiz results taken before account creation MUST be stored with a session or token identifier, not tied to any user account.

#### 5.1.3 Pre-to-Post-Account Result Transfer

- When a user creates an account after taking the quiz, the platform MUST offer to import their quiz results.
- Imported results SHOULD pre-populate the methodology selection wizard in onboarding.
- The transfer mechanism MUST work across browser sessions (e.g., via a result URL or code the user can enter).

### 5.2 Methodology Explorer Pages `[V§6]`

- The platform MUST provide a dedicated page for each supported methodology.
- Each methodology page MUST include: philosophy and history, what a typical day looks like, pros and cons, recommended starting resources, and real family stories.
- Methodology explorer pages MUST be publicly accessible and SEO-indexable.
- Content MUST be detailed enough for a parent to confidently choose an approach. `[V§6]`
- Pages SHOULD include links to related marketplace content and community groups (visible but requiring account to access).

### 5.3 State Legal Guides `[V§6]`

- The platform MUST provide homeschooling legal guides for all 50 US states plus the District of Columbia.
- Each guide MUST include: notification/registration requirements, required subjects, assessment/testing obligations, record-keeping requirements, attendance requirements, and umbrella school options (where applicable).
- Guides MUST be publicly accessible and SEO-indexable.
- Each guide MUST display a "last reviewed" date and a legal disclaimer that the content is informational, not legal advice.
- Guides MUST follow a consistent structured data format to enable programmatic querying (e.g., for compliance tool configuration).
- The platform MUST define an editorial workflow for guide updates: drafting, legal review, publication, and scheduled re-review (at minimum annually).

### 5.4 Homeschooling 101 & Advocacy Content `[V§6]`

- The platform MUST provide educational content addressing common concerns: socialization, cost, time commitment, dual-income families, special needs, transitioning mid-year, and responding to skepticism.
- The platform MUST provide "case for homeschooling" content that is confident and clear-eyed, not defensive. `[V§6]`
- All Discovery content MUST contain zero user-generated content and zero personal data. `[V§7]`

### 5.5 SEO & Public Content Strategy

- All public discovery content MUST be server-rendered or pre-rendered for search engine indexability.
- State legal guides SHOULD target "[state] homeschool requirements" and related search queries. `[V§6]`
- The methodology quiz SHOULD be designed for social sharing and word-of-mouth distribution.
- Public content pages MUST include appropriate structured data markup (e.g., FAQ schema, breadcrumbs).

---

## 6. Onboarding & Family Setup `[V§6]`

### 6.1 Account Creation Flow

- The platform MUST support account creation via email/password and at least one OAuth provider.
- The account creation flow MUST display a COPPA compliance notice informing users that the platform handles children's data and explaining parental consent requirements. `[V§7]`
- Account creation MUST create both a Parent User and a Family Account atomically.
- Email verification MUST be required before the account is fully activated.

### 6.2 Family Profile Setup

- The onboarding flow MUST collect: parent name(s), child information (name, age/birth year, grade level for each child), and family location (state, for compliance purposes).
- The flow SHOULD allow adding multiple children during initial setup.
- The flow SHOULD allow adding a co-parent during or after initial setup.
- Location collection MUST explain why it is needed (state-specific legal information and local community discovery).

### 6.3 Methodology Selection Wizard

- If the user has prior quiz results, the wizard MUST offer to import them as a starting point.
- The wizard MUST support three paths:
  1. **Quiz-informed**: Pre-populated from quiz results, with ability to adjust.
  2. **Exploration mode**: Browse methodology explorer summaries and select directly.
  3. **"I don't know yet"**: Select a default or skip, with a prompt to revisit later.
- The wizard MUST allow selecting a primary methodology and optional secondary methodologies.
- The wizard MUST clearly explain the multi-methodology model: what primary vs. secondary means, how tools are combined, and that selections can be changed at any time.

### 6.4 Getting-Started Roadmap `[V§6]`

- After methodology selection, the platform MUST present a methodology-specific getting-started roadmap.
- The roadmap MUST be a concrete, actionable first-week checklist — not generic advice.
- Roadmaps MUST be age-adapted: a Charlotte Mason roadmap for a family with a 6-year-old SHOULD differ from one for a family with a 12-year-old.
- Roadmap items SHOULD link to relevant platform features, methodology education content, and marketplace recommendations.

### 6.5 Starter Curriculum Recommendations `[V§6]`

- The platform MUST present curated starter curriculum recommendations drawn from the marketplace.
- Recommendations MUST be specific and limited (e.g., "the 3 most popular Charlotte Mason starter packages for a 2nd grader"), not an overwhelming catalog dump. `[V§6]`
- Recommendations MUST be methodology-specific, age-appropriate, and informed by community ratings.
- Recommendations MUST include both free and paid options.

### 6.6 Community Connections `[V§6]`

- The onboarding flow MUST suggest methodology-matched groups.
- The flow SHOULD suggest nearby homeschool families (if location is provided and matches exist).
- The flow SHOULD suggest mentor matching with experienced homeschoolers who use the same methodology.
- All suggestions MUST respect the privacy model — the user opts in to visibility, not the suggested connections.

---

## 7. Social Layer `[V§7]`

### 7.1 Family Profiles

- Every family MUST have a profile visible to friends.
- Profiles MUST include: family display name, parent name(s), children's first names and ages, location (granularity chosen by user), and selected methodology(ies).
- Each profile field MUST support per-field privacy controls with a minimum of: **friends-only** (default) and **hidden**.
- Profile photos MAY be supported for the family and individual parents.
- Student photos MUST NOT be required and MUST default to a generic avatar.
- There MUST NOT be public profiles — all profile data is visible only to friends by default. `[V§7, V§12]`

### 7.2 Timeline / Feed

#### 7.2.1 Post Types

The feed MUST support the following post types:

| Post Type | Description | Attachments |
|-----------|-------------|-------------|
| **Text** | Free-form text update | Optional images |
| **Photo** | Image-focused post | Required image(s), optional caption |
| **Learning Milestone** | Auto-generated or manual milestone (e.g., "completed unit study") | Auto-populated from learning tools |
| **Event Share** | Shared event with RSVP link | Links to event entity |
| **Marketplace Review** | Shared review of purchased content | Links to marketplace listing |
| **Resource Share** | Link to external or marketplace content | URL or listing reference |

#### 7.2.2 Visibility

- All posts MUST default to **friends-only** visibility. `[V§7]`
- Posts within groups MUST be visible to group members only.
- There MUST NOT be a "public" visibility option for any user-generated content. `[V§7, V§12]`

#### 7.2.3 Feed Algorithm

- The feed MUST use **reverse-chronological ordering** for MVP.
- Algorithmic ranking MAY be introduced post-MVP, but MUST include a user toggle to revert to chronological.
- The feed MUST NOT display paid/sponsored content without clear labeling. `[V§10]`

### 7.3 Comments

- Comments MUST support threading (at least one level of reply depth).
- Comments MUST inherit the visibility of the parent post.
- Comment authors MUST be able to delete their own comments.
- Post authors MUST be able to delete any comment on their post.

### 7.4 Friend System

- Friendships MUST be bidirectional (require mutual acceptance of a friend request).
- Users MUST be able to discover potential friends through: methodology-matched suggestions, group membership, location-based discovery, and search by name.
- Users MUST be able to block other users. Blocking MUST be silent (the blocked user is not notified) and MUST prevent all interaction including viewing the blocker's profile.
- Unfriending MUST be available without notification to the other party.

### 7.5 Direct Messaging

- Direct messaging MUST be limited to **parent-to-parent** communication only. `[V§7]`
- For MVP, direct messaging MUST be limited to friends only.
- Messages MUST support text and image attachments.
- Users MUST be able to report messages for moderation review.
- Message history MUST be retained and accessible to both participants until one party deletes the conversation (for their view only).

### 7.6 Groups `[V§7]`

- The platform MUST create and maintain **platform-managed groups** for each supported methodology (e.g., "Charlotte Mason Community," "Unschooling Families").
- Users MUST be able to create **user-created groups** with customizable name, description, and membership rules.
- Group types MUST include at minimum: open (anyone can join), request-to-join (moderator approval), and invite-only.
- Each group MUST have at least one moderator (the creator by default).
- Group posts MUST be visible only to group members.
- Groups SHOULD support pinned posts and group-specific events.

### 7.7 Events `[V§7]`

- Any user MUST be able to create an event with: title, description, date/time, location (physical or virtual), and capacity (optional).
- Events MUST support RSVP (going, interested, not going).
- Events MUST support **recurring events** (weekly, biweekly, monthly patterns).
- Events MUST be discoverable by: location (nearby), group membership, and methodology.
- Event visibility MUST follow the creator's choice: friends-only, group-only, or discoverable by location/methodology.
- Events MUST support cancellation with automatic notification to RSVPed attendees.

### 7.8 Location-Based Discovery `[V§7]`

- Location-based features MUST be **opt-in** — users choose to share their location.
- Location granularity MUST be **coarse-grained** (city or region level), not precise coordinates.
- Location-based discovery MUST include: nearby families, nearby groups, and nearby events.
- Users MUST be able to disable location sharing at any time, immediately removing them from location-based results.
- The platform MUST NOT store precise geolocation data, even temporarily.

---

## 8. Learning Layer `[V§8]`

### 8.1 Tool Catalog

The master tool catalog defines every learning tool available on the platform. Each tool has a unique identifier, belongs to one or more methodology activations (see §4.2), and MAY be gated by subscription tier (free or premium).

#### 8.1.1 Activities (All Methodologies)

- Users MUST be able to log daily learning activities with: title, description (optional), subject/skill tags, student(s), date, and duration (optional).
- Activities MUST support attachments (photos, files).
- Activities MUST be filterable by student, subject, date range, and methodology.
- **Tier**: Free

#### 8.1.2 Tests & Grades (Traditional, Classical)

- Users MUST be able to record assessments with: title, subject, student, date, score (percentage, letter grade, or points), and weight (optional).
- The system MUST calculate running averages per subject per student.
- Users MUST be able to define grading scales (e.g., A/B/C/D/F, pass/fail, custom).
- **Tier**: Free (basic), Premium (advanced analytics)

#### 8.1.3 Reading Lists (Charlotte Mason, Classical)

- Users MUST be able to create and manage book lists with: title, author, subject/methodology tags, and status (to-read, in-progress, completed).
- The system SHOULD support importing books by ISBN or title search.
- Completed books SHOULD auto-generate an activity log entry.
- Users SHOULD be able to share reading lists with friends or groups.
- **Tier**: Free

#### 8.1.4 Journaling & Narration (Charlotte Mason, Waldorf, Unschooling)

- Users MUST be able to create journal entries with: text content, date, student, subject/activity tags, and optional image attachments.
- The system MUST support entry types: **free-form journal**, **narration record** (Charlotte Mason), and **reflection/documentation** (Unschooling).
- Entries MUST be searchable by keyword and filterable by student, date, and type.
- **Tier**: Free (basic), Premium (enhanced storage)

#### 8.1.5 Projects (Waldorf, Montessori, Unschooling)

- Users MUST be able to create multi-step projects with: title, description, student(s), subject/skill tags, and milestones.
- Each milestone MUST support: name, target date (optional), completion status, and notes/attachments.
- Projects MUST support a status lifecycle: planning → in-progress → completed.
- Completed projects SHOULD be exportable as portfolio entries.
- **Tier**: Free (basic), Premium (portfolio integration)

#### 8.1.6 Video Lessons (Marketplace Integration)

- Purchased video content from the marketplace MUST be accessible through the learning tools interface.
- Video lesson progress (watched/unwatched, timestamp) MUST be tracked per student.
- Video lessons MUST be loggable as activities with auto-populated metadata from the marketplace listing.
- **Tier**: Free (marketplace purchase required)

#### 8.1.7 Progress Tracking & Analytics

- **Free tier**: Basic progress tracking — activity counts, reading list completion, and simple subject-hours-per-week summaries.
- **Premium tier**: Advanced analytics — trend visualization over time, subject balance analysis, comparison to methodology-typical benchmarks (not standardized testing benchmarks), and exportable progress reports.
- Progress data MUST be per-student.
- Progress views MUST support date range filtering.
- **Tier**: Free (basic), Premium (advanced)

#### 8.1.8 Methodology-Specific Tools

The following tools are activated only for specific methodologies:

| Tool | Methodology | Description |
|------|-------------|-------------|
| **Nature Journals** | Charlotte Mason | Specialized journal for nature observations with drawing/photo support and species identification fields |
| **Trivium Tracker** | Classical | Track student progress through grammar, logic, and rhetoric stages per subject |
| **Rhythm Planner** | Waldorf | Weekly/daily rhythm planning with blocks for main lessons, artistic activities, practical work, and free play |
| **Observation Logs** | Montessori | Structured observation records: work chosen, duration, concentration level, social interactions, follow-up notes |
| **Habit Tracking** | Charlotte Mason | Track daily/weekly habit formation goals with streaks and parent notes |
| **Interest-Led Activity Log** | Unschooling | Document child-initiated learning with auto-tagging to subjects/skills |
| **Handwork Project Tracker** | Waldorf | Track handwork and craft projects with materials, techniques, and photos |
| **Practical Life Activities** | Montessori | Log and track practical life skill development with age-appropriate activity suggestions |

- All methodology-specific tools MUST follow the same data patterns as core tools (taggable, searchable, exportable, per-student).
- **Tier**: MVP methodology-specific tools are Free (basic); advanced features are Premium.

### 8.2 Per-Student Tool Assignment

- When a family has multiple methodologies selected, parents MUST be able to assign specific tools to specific students (e.g., one child uses Charlotte Mason tools, another uses Traditional tools).
- Tool assignment MUST default to the family's methodology-derived tool set unless overridden.
- Parents MUST be able to manually activate or deactivate individual tools per student, regardless of methodology selection.

### 8.3 Subject / Skill Taxonomy

- The platform MUST define a hierarchical subject taxonomy (e.g., Math → Algebra → Linear Equations; Language Arts → Writing → Creative Writing).
- The taxonomy MUST be extensible — administrators MUST be able to add new subjects and skills without code changes.
- Users MUST be able to tag activities, assessments, and projects with subjects from the taxonomy.
- Users SHOULD be able to create custom subjects/skills within their family's scope.
- The taxonomy MUST be consistent across learning tools, marketplace content tagging, and compliance reporting.

### 8.4 Tool-Adjacent Parent Education `[V§8]`

- Learning tools MUST surface methodology-specific guidance contextually (e.g., "How to use narration effectively" within the narration tool). `[V§8]`
- Guidance content MUST be sourced from the Methodology domain's philosophy modules and mastery paths.
- Guidance MUST NOT block or interrupt the user's workflow — it SHOULD be accessible via a help panel, tooltip, or expandable section.

### 8.5 Data Ownership & Portability

- All learning data entered by a family MUST be owned by that family.
- Families MUST be able to export all their learning data in a standard, machine-readable format (e.g., CSV, JSON).
- Export MUST include: activities, assessments, reading lists, journal entries, projects, and progress records.
- Export MUST be available at any time, regardless of subscription tier.
- Data export MUST be completable within a reasonable timeframe (SHOULD be available for download within 24 hours of request).

---

## 9. Curriculum Marketplace `[V§9]`

### 9.1 Creator Onboarding

- Creators MUST provide: legal name, contact email, and agreement to marketplace terms of service.
- Before receiving payouts, creators MUST complete identity verification and provide tax information (SSN/EIN for US 1099-K compliance).
- Creators MUST set up a store profile: display name, bio, and optional logo/banner.
- The onboarding process MUST clearly communicate the revenue share model, payout schedule, and content policies.

### 9.2 Content Listings

#### 9.2.1 Listing Fields

| Field | Required | Description |
|-------|:--------:|-------------|
| Title | ✓ | Content title |
| Description | ✓ | Detailed description with formatting support |
| Price | ✓ | Price in USD (or free) |
| Methodology tags | ✓ | One or more applicable methodologies |
| Subject tags | ✓ | One or more subjects from the platform taxonomy |
| Grade/age range | ✓ | Target audience age or grade range |
| Content type | ✓ | Category (curriculum package, worksheet, unit study, video, etc.) |
| Worldview tags | | Creator self-reported worldview categorization (e.g., secular, Christian, Jewish, Islamic, neutral) `[V§12]` |
| Preview content | | Free preview or sample pages |
| Thumbnail image | | Display image for search results |
| File(s) | ✓ | The digital content files (PDF, video, etc.) |

#### 9.2.2 Listing States

Listings MUST follow a lifecycle: **Draft** → **Submitted** → **Published** → **Archived**.

- **Draft**: Creator is editing; not visible to buyers.
- **Submitted**: Awaiting automated content screening (not full editorial review). `[V§9]`
- **Published**: Live and purchasable.
- **Archived**: Removed from discovery but still accessible to existing purchasers.

#### 9.2.3 Versioning

- Creators MUST be able to update published listings (description, price, files).
- File updates MUST be reflected for existing purchasers (they receive the updated version).
- Price changes MUST NOT affect existing purchases.
- The system SHOULD maintain a version history for creator reference.

### 9.3 Discovery & Search

- The marketplace MUST support faceted filtering by: methodology, subject, grade/age range, content type, price range (including free), worldview tags, and rating.
- The marketplace MUST support full-text search across listing titles and descriptions.
- The marketplace MUST feature curated sections: **Featured**, **Trending**, **New Arrivals**, and **Staff Picks**.
- The marketplace SHOULD support methodology-scoped browsing (e.g., "Charlotte Mason → Language Arts → Grade 3").
- Search results MUST be sortable by: relevance, price (low-high, high-low), rating, and recency.

### 9.4 Purchase Flow

- The marketplace MUST support a cart-based purchase flow (add to cart → checkout → payment → access).
- The marketplace MUST support free content (zero-price listings) with a simplified "Get" flow (no payment required).
- The marketplace MUST support content bundles (multiple listings sold together at a combined price).
- After purchase, content MUST be immediately accessible in the buyer's account.
- The marketplace MUST provide purchase receipts via email.
- Purchased content MUST be accessible regardless of subscription tier. `[V§10]`

### 9.5 Ratings & Reviews `[V§9]`

- Only users who have purchased content MAY leave a rating or review (**verified-purchaser only**). `[V§9]`
- Ratings MUST use a numerical scale (e.g., 1-5 stars).
- Reviews MUST support text feedback.
- Reviews MUST be anonymous by default (reviewer identity not shown to the creator or other users). `[V§9]`
- Reviews MUST be subject to content moderation (see §12).
- Creators MUST be able to respond to reviews (publicly visible response).
- The platform MUST display aggregate rating and review count on listing pages.

### 9.6 Revenue Share & Payouts `[V§9, V§10]`

- The platform MUST implement a revenue share model where creators receive the majority of each sale.
- The specific revenue share percentage is an **open question** (see §20) — recommended range is 70-75% to creator.
- Creators MUST have access to a creator dashboard showing: sales history, earnings, pending payouts, and analytics (views, conversion rate).
- Payouts MUST follow a defined schedule (e.g., monthly, with a minimum payout threshold).
- The platform MUST handle refund deductions from creator earnings.
- The platform MUST issue 1099-K forms to creators meeting IRS thresholds. `[V§10]`

### 9.7 Integration with Learning Tools `[V§9]`

- Purchased curriculum content MUST be accessible from within the family's learning tools.
- When a family purchases content tagged with a methodology they use, the platform SHOULD prompt integration with relevant tools (e.g., adding books to reading lists, adding activities to the planner).
- AI recommendations MUST draw from the marketplace catalog. `[V§8]`

---

## 10. AI & Recommendations `[V§8]`

### 10.1 Recommendation Engine

- The recommendation engine MUST consider the following inputs: family's selected methodology(ies), student age(s) and grade level(s), learning history and progress data, marketplace purchase history, and community signals (popular content among similar families).
- Recommendations MUST be constrained by the family's methodology selection — a Charlotte Mason family SHOULD NOT receive Traditional textbook recommendations unless they explicitly browse outside their methodology.
- Recommendation outputs MUST include: marketplace content suggestions, activity ideas, reading suggestions, and community/group suggestions.
- Recommendations MUST be a **premium feature**. `[V§10]`

### 10.2 Content Suggestions

- The system SHOULD surface content suggestions based on: community popularity among methodology-matched families, seasonal appropriateness (e.g., nature study topics by season), progress gaps (e.g., if a student hasn't had math activities in two weeks), and student age/grade transitions (e.g., approaching a new trivium stage).
- Content suggestions MUST clearly indicate the source signal (e.g., "Popular with Charlotte Mason families," "Based on your reading history").

### 10.3 AI Tutoring — Future Scope

- AI tutoring is **out of scope** for this specification (see §1.5).
- However, the platform SHOULD collect and store anonymized, aggregated learning interaction data that can later be used to train or inform AI tutoring capabilities.
- Data collection MUST comply with the privacy model (§17.2) and MUST NOT include personally identifiable information in aggregated datasets.

### 10.4 Ethical AI Considerations `[V§12]`

- The recommendation engine MUST NOT create filter bubbles — it SHOULD periodically surface content outside the user's typical patterns with clear labeling.
- Recommendation logic SHOULD be transparent — users SHOULD understand why a recommendation was made.
- Parent authority MUST be respected — parents MUST be able to dismiss, block, or adjust recommendations. `[V§12]`
- The recommendation engine MUST maintain content neutrality — it MUST NOT favor or suppress content based on worldview, religious affiliation, or methodology preference beyond the user's own selections. `[V§12]`
- AI-generated suggestions MUST be clearly labeled as AI-generated.

---

## 11. Compliance & Reporting `[V§8]`

### 11.1 State-Specific Compliance Configuration

- The platform MUST maintain a structured compliance configuration for each US state (50 states + DC), defining: required subjects, assessment types, notification/filing requirements, attendance requirements, and record-keeping obligations.
- Compliance configuration MUST be derived from the same source data as the public state legal guides (§5.3) and MUST be kept in sync.
- Compliance features are **premium only**. `[V§10]`

### 11.2 Attendance Logging

- The platform MUST support daily attendance marking per student.
- Attendance records MUST support statuses: present (full day), present (partial), absent, and not-applicable (e.g., weekends, breaks).
- The system MUST generate attendance summaries by date range, meeting state-specific minimum day/hour requirements.
- The platform SHOULD support custom schedule definitions (e.g., 4-day weeks, year-round schooling).

### 11.3 Assessment Records

- The platform MUST aggregate assessment data from learning tools (tests, grades, project completions) into compliance-ready records.
- Assessment records MUST be filterable by student, subject, and date range.
- The platform SHOULD support standardized test score entry (for states that require standardized testing).

### 11.4 Portfolio Generation

- The platform MUST generate downloadable portfolios in PDF format.
- Portfolios MUST be customizable: parents select which learning artifacts (activities, journal entries, projects, assessments, reading lists) to include.
- Portfolios MUST be organizable by subject, chronologically, or by student.
- Portfolios MUST include a cover page with student and family information.
- Portfolio generation is a **premium feature**. `[V§10]`

### 11.5 Transcript Generation

- The platform MUST generate formal academic transcripts for high school students (grades 9-12).
- Transcripts MUST include: student name, courses/subjects, grades, credit hours, and cumulative GPA.
- Transcripts MUST follow a format accepted by colleges and universities (standard high school transcript format).
- The platform MUST support multiple GPA calculation methods (4.0 scale, weighted, unweighted).
- Transcript generation is a **premium feature**. `[V§10]`

### 11.6 Record Retention Policies

- Learning records MUST be retained for the lifetime of the account.
- Upon account deletion, learning records MUST be exportable (§8.5) before permanent deletion.
- After the export/grace period, deletion MUST be permanent and irreversible.
- Records required for financial/legal compliance (marketplace transactions, tax records) MUST be retained per applicable legal requirements regardless of account status.

---

## 12. Content Moderation & Trust and Safety `[V§7]`

### 12.1 CSAM Detection & Response `[V§7]`

- The platform MUST implement automated CSAM detection on all user-uploaded images and videos using industry-standard tools (e.g., PhotoDNA, NCMEC hash matching).
- Confirmed or suspected CSAM MUST be immediately removed from the platform.
- The platform MUST report confirmed CSAM to the National Center for Missing & Exploited Children (NCMEC) as required by federal law (18 U.S.C. § 2258A).
- The platform MUST preserve evidence as required by law enforcement.
- The platform MUST NOT notify the offending user of the detection or report.
- Accounts associated with CSAM MUST be immediately and permanently suspended.
- This is a **zero-tolerance** policy. `[V§7]`

### 12.2 Content Moderation Pipeline

The moderation pipeline MUST follow this sequence:

1. **Automated Screening** — All user-generated content (posts, comments, messages, reviews, marketplace listings) MUST pass through automated screening for: CSAM (§12.1), explicit/adult content, spam, and prohibited content.
2. **Community Reporting** — Users MUST be able to report content through the reporting system (§12.3).
3. **Human Review** — Flagged content (automated or reported) MUST be routed to human moderators for review.
4. **Actions** — Moderators MUST be able to: remove content, issue warnings, temporarily suspend accounts, and permanently ban accounts.
5. **Appeals** — Users who receive moderation actions MUST have a mechanism to appeal. Appeals MUST be reviewed by a different moderator than the one who took the original action.

### 12.3 User Reporting System

- Users MUST be able to report: posts, comments, messages, profiles, groups, events, marketplace listings, and reviews.
- Report categories MUST include at minimum: inappropriate content, harassment/bullying, spam, misinformation, CSAM/child safety, and "other" (with free-text description).
- Reports MUST be triaged by priority: child safety reports MUST be reviewed within 24 hours; other reports SHOULD be reviewed within 72 hours.
- Reporters MUST receive acknowledgment of their report and notification of the outcome (without revealing details of actions taken against the reported user).

### 12.4 Bot Prevention `[V§7]`

- Account creation MUST include CAPTCHA or equivalent bot detection.
- The platform MUST implement behavioral detection for bot-like activity patterns (e.g., rapid posting, mass friend requests, repetitive content).
- Rate limiting MUST be enforced on all user actions (posting, messaging, friend requests, etc.).
- Detected bot accounts MUST be flagged for review and MAY be automatically suspended pending review.

### 12.5 Inter-Methodology Hostility `[V§7]`

- The platform MUST establish clear community guidelines prohibiting: attacks on families based on their methodology choice, religious vs. secular hostility, and unsolicited criticism of another family's educational approach. `[V§7]`
- Methodology-related hostility reports MUST be a distinct category in the reporting system.
- Platform-managed methodology groups MUST have dedicated moderators aware of this policy.

### 12.6 Child Safety Beyond CSAM

- The platform MUST implement monitoring for grooming behavior patterns in messaging and comments.
- The platform MUST enforce age-appropriate content filtering for any content surfaces accessible to Student Profiles.
- The platform MUST prohibit adults from initiating direct contact with Student Profiles.
- Parental controls MUST allow parents to restrict their students' content exposure beyond platform defaults.

### 12.7 Moderation Tooling

- Platform administrators MUST have access to a moderation dashboard with: content review queue (prioritized), user account management, moderation action history, and report analytics.
- All moderation actions MUST be audit-logged with: moderator identity, timestamp, action taken, and rationale.
- The moderation system MUST support moderator roles for user-created groups (group-level moderation, distinct from platform-level).

---

## 13. Notifications & Communication

### 13.1 Notification Types

| Category | Examples |
|----------|---------|
| **Social** | Friend requests, new messages, post comments, group invitations, event invitations/reminders |
| **Learning** | Milestone completions, streak achievements, methodology mastery path progress, reminder prompts |
| **Marketplace** | Purchase confirmations, new reviews on creator's content, content updates, sale announcements |
| **System** | Account security alerts, subscription changes, policy updates, moderation actions |

### 13.2 Delivery Channels

- **In-app**: All notification types MUST be delivered in-app through a notification center.
- **Email**: Social, marketplace, and system notifications MUST be deliverable via email (user-configurable).
- **Push notifications**: MAY be supported in future phases (mobile apps). The notification system MUST be architected to support additional channels without redesign.

### 13.3 User Preferences

- Users MUST be able to configure notification preferences per type and per channel (e.g., enable email for friend requests, disable email for post comments).
- Users MUST be able to opt out of all non-essential email notifications.
- The platform MUST support email digest options: immediate, daily digest, weekly digest, or off.
- All marketing/promotional emails MUST comply with CAN-SPAM requirements (unsubscribe link, physical address, honest subject lines).
- System-critical notifications (security alerts, moderation actions) MUST NOT be disableable.

---

## 14. Search & Discovery (Internal)

### 14.1 Search Scopes

The internal search system MUST support the following scopes:

| Scope | Searchable Content |
|-------|-------------------|
| **Social** | Users (by name), groups (by name/description), events (by title/description/location) |
| **Learning** | Activities, journal entries, reading lists, projects (within the family's own data) |
| **Marketplace** | Listings (by title, description, creator, tags) |

### 14.2 Search Requirements

- Search results MUST be returned in sub-second time (< 500ms p95) for common queries.
- Search MUST support **autocomplete** / type-ahead suggestions.
- Marketplace search MUST support **faceted filtering** (methodology, subject, grade, price, rating, content type, worldview).
- Social search MUST respect privacy — users MUST NOT appear in search results for users who are not their friends, unless they have opted into discovery (§7.8).
- Learning search is scoped to the authenticated family's own data — cross-family learning data MUST NOT be searchable.

### 14.3 Discovery Features

- The platform MUST provide methodology-scoped "Explore" sections for social (families/groups/events) and marketplace (content) discovery.
- Discovery sections SHOULD surface content based on methodology match, location proximity (opt-in), popularity, and recency.
- Discovery MUST respect all privacy settings — only users who have opted into discovery features appear in discovery results.

---

## 15. Revenue, Subscriptions & Billing `[V§10]`

### 15.1 Free Tier

The following features MUST be available to all users without payment: `[V§10]`

- Full social features (profiles, timeline, comments, friends, messaging, groups, events, location discovery)
- Basic methodology-scoped learning tools (Activities, Reading Lists, Journaling & Narration, basic Progress Tracking)
- Marketplace access (browse, purchase, rate/review)
- Methodology education modules (philosophy, getting-started, basic mastery path)
- Discovery tools (quiz, methodology explorer, state legal guides, Homeschooling 101)
- Full onboarding flow (family setup, methodology wizard, getting-started roadmap)
- Data export

The free tier MUST be "genuinely useful — not a crippled experience." `[V§10]`

### 15.2 Premium Tier

The premium subscription adds: `[V§10]`

- State compliance reporting (attendance, assessment records)
- Portfolio generation (PDF export)
- Transcript generation (high school)
- Advanced progress analytics and insights
- AI-powered content recommendations
- Enhanced storage for journals, projects, and portfolios
- Advanced methodology mastery paths and personalized growth recommendations
- Methodology-specific advanced tools (as designated in §8.1.8)
- Additional premium features as defined in future specifications

### 15.3 Subscription Management

- Subscriptions MUST be at the **family level**, not per-student. `[V§10]`
- The platform MUST support monthly and annual billing cycles.
- Annual billing SHOULD offer a discount over monthly.
- Upgrades MUST take effect immediately with prorated billing.
- Downgrades MUST take effect at the end of the current billing period.
- On downgrade from premium to free:
  - All learning data MUST be preserved (not deleted).
  - Premium tools MUST become read-only (historical data viewable, no new entries).
  - Compliance reports and portfolios already generated MUST remain downloadable.
  - AI recommendations MUST be disabled.
- Subscription cancellation MUST provide a clear data preservation guarantee.
- The platform MUST send advance notice before subscription renewal.

### 15.4 Marketplace Transactions

- Payment processing MUST be handled through a third-party payment processor.
- The platform MUST support standard payment methods (credit/debit cards; additional methods MAY be added).
- Marketplace purchases MUST be separate from subscription billing.
- The platform MUST handle sales tax collection and remittance per applicable jurisdictions.
- The platform MUST issue 1099-K forms to creators meeting IRS reporting thresholds.
- **Refund policy**: The platform MUST define and enforce a refund policy for marketplace purchases. The recommended policy is a 30-day satisfaction guarantee for first-time purchases of a listing, with creator agreement.

---

## 16. Data Architecture (Conceptual)

### 16.1 Core Entities & Relationships

```
Family Account
├── Parent User (1..n)
│   ├── Credentials (email/OAuth)
│   ├── Social Profile
│   ├── Notification Preferences
│   └── Creator Account (0..1)
│       ├── Store Profile
│       ├── Marketplace Listings (0..n)
│       └── Payout Configuration
├── Student Profile (0..n)
│   ├── Demographics (name, age, grade)
│   ├── Methodology Override (0..1)
│   └── Learning Data
│       ├── Activities (0..n)
│       ├── Assessments (0..n)
│       ├── Journal Entries (0..n)
│       ├── Projects (0..n)
│       ├── Reading Lists (0..n)
│       └── Progress Records
├── Methodology Selection
│   ├── Primary Methodology (1)
│   └── Secondary Methodologies (0..n)
├── Subscription (0..1)
├── Purchases (0..n) → Marketplace Listing
└── Social Connections
    ├── Friendships (0..n) → Family Account
    ├── Group Memberships (0..n) → Group
    └── Event RSVPs (0..n) → Event

Methodology (platform-defined)
├── Identity (name, description, icon)
├── Philosophy Module
├── Tool Registry → Learning Tool (n..m)
├── Content Tags
├── Onboarding Config
├── Mastery Paths
└── Community Config

Learning Tool (platform-defined)
├── Identity (name, description)
├── Configuration Schema
└── Methodology Activations (n..m) → Methodology

Group
├── Metadata (name, description, type)
├── Moderators (1..n) → Parent User
├── Members (0..n) → Parent User
├── Posts (0..n)
└── Events (0..n)

Event
├── Metadata (title, date, location, recurrence)
├── Creator → Parent User
├── RSVPs (0..n) → Parent User
└── Group (0..1) → Group

Marketplace Listing
├── Metadata (title, description, price, tags)
├── Creator → Creator Account
├── Files (1..n)
├── Ratings (0..n)
├── Reviews (0..n)
└── Version History
```

### 16.2 Key Data Invariants

- **Ownership**: All learning data MUST be owned by the Family Account that created it. No cross-family data access is permitted except through explicit social sharing mechanisms (e.g., shared reading lists).
- **Privacy**: All user-generated content MUST default to friends-only visibility. No data path MAY expose user content to unauthenticated users.
- **Attribution**: Marketplace content MUST always maintain association with its creator, even if the creator account is deactivated.
- **Referential Integrity**: Deleting a parent entity (e.g., Family Account) MUST cascade appropriately — learning data deleted, social connections severed, marketplace purchases retained (for financial compliance).

### 16.3 Data Lifecycle

| Event | Action |
|-------|--------|
| **Account creation** | Family, parent, and initial configuration created atomically. |
| **Student profile deletion** | Learning data exportable → grace period → permanent deletion. |
| **Family account deletion** | Full data export offered → active subscriptions cancelled → grace period (minimum 30 days) → permanent deletion of all family data. Marketplace purchase records retained per legal requirements. |
| **COPPA deletion request** | Parent requests deletion of child data → MUST be processed within regulatory timeframe. |
| **Creator account deactivation** | Published listings MAY remain accessible to existing purchasers. New purchases MUST be disabled. Pending payouts MUST be settled. |
| **Content removal (moderation)** | Content removed from public access. Original data retained for moderation audit trail per retention policy. |

---

## 17. Non-Functional Requirements

### 17.1 Security

- Authentication MUST support multi-factor authentication (MFA) as an option for all users and SHOULD encourage it during onboarding.
- All data in transit MUST be encrypted using TLS 1.2 or higher.
- All sensitive data at rest (credentials, payment information, PII) MUST be encrypted.
- API endpoints MUST implement authentication, authorization, input validation, and rate limiting.
- The platform MUST undergo penetration testing before public launch and at least annually thereafter.
- Session management MUST enforce reasonable timeouts and support remote session revocation (e.g., "log out all devices").
- The platform MUST maintain a vulnerability disclosure program.

### 17.2 Privacy `[V§7, V§12]`

#### COPPA Compliance Checklist `[V§7]`

The platform MUST comply with the Children's Online Privacy Protection Act:

- [ ] Obtain verifiable parental consent before collecting personal information from children under 13.
- [ ] Provide clear, comprehensive privacy notice describing data practices for children's information.
- [ ] Allow parents to review, modify, and delete their child's personal information.
- [ ] Limit data collection from children to what is reasonably necessary.
- [ ] Maintain reasonable security measures for children's data.
- [ ] Do NOT condition a child's participation on providing more information than necessary.
- [ ] Do NOT share children's personal information with third parties except as necessary for platform operation.

#### Additional Privacy Requirements

- Data minimization: The platform MUST NOT collect data beyond what is necessary for the stated purpose.
- Users MUST have the ability to export all their data (see §8.5).
- Users MUST have the ability to request complete account deletion (see §16.3).
- The platform SHOULD be architected for GDPR compliance readiness, even though the initial launch is US-only. `[V§3]`
- The privacy policy MUST be written in clear, accessible language.
- Third-party data sharing MUST be limited to what is necessary for platform operation (e.g., payment processing) and MUST be disclosed in the privacy policy.

### 17.3 Performance Targets

| Metric | Target |
|--------|--------|
| Page load (initial) | < 3 seconds (p95) |
| Page load (subsequent / SPA navigation) | < 1 second (p95) |
| API response (standard) | < 300ms (p95) |
| API response (complex queries) | < 1 second (p95) |
| Search results | < 500ms (p95) |
| Concurrent users (launch) | 10,000 simultaneous |
| Concurrent users (scale target) | 100,000 simultaneous |

### 17.4 Scalability

- The platform MUST be designed for horizontal scaling — adding capacity by adding instances, not upgrading hardware.
- Media storage (images, videos, files) MUST be handled by a dedicated, independently scalable storage service.
- Search indexing MUST be decoupled from primary data storage.
- The system SHOULD support independent scaling of read-heavy domains (Social, Marketplace browse) vs. write-heavy domains (Learning, Activity logging).

### 17.5 Availability

- Target uptime: **99.9%** (approximately 8.7 hours downtime per year).
- The platform MUST implement graceful degradation — if a non-critical domain (e.g., AI Recommendations) is unavailable, core domains (Social, Learning, Marketplace) MUST continue to function.
- Planned maintenance MUST be scheduled during low-traffic periods with advance notice to users.
- The platform MUST implement automated health monitoring and alerting.

### 17.6 Accessibility

- The platform MUST conform to **WCAG 2.1 Level AA** accessibility standards.
- All interactive elements MUST be keyboard-navigable.
- All images MUST have appropriate alt text.
- Color MUST NOT be the sole means of conveying information.
- The platform MUST support screen readers.
- Form fields MUST have associated labels.
- The platform SHOULD conduct accessibility audits before each major release.

### 17.7 Internationalization Readiness

- The initial launch is **US-only**. `[V§3]`
- However, the platform MUST be architected for future internationalization:
  - All user-facing strings MUST be externalized (not hardcoded).
  - Date, time, and number formatting MUST use locale-aware formatting.
  - The data model MUST support multi-currency (for future marketplace expansion).
  - The subject taxonomy MUST accommodate different national curriculum standards.

### 17.8 Browser & Device Support

- The platform MUST support the latest two major versions of: Chrome, Firefox, Safari, and Edge. `[V§11]`
- The platform MUST be fully responsive from 320px (mobile) to 2560px (ultrawide) viewport widths. `[V§11]`
- Touch interactions MUST be supported for tablet and mobile browsers.
- The platform MUST NOT require browser plugins or extensions.

---

## 18. Integration Points & Cross-Domain Contracts

This section defines the key data flows between domains. These contracts are logical — implementation details (synchronous vs. asynchronous, API vs. event) are deferred to architecture design.

### 18.1 Methodology → All Domains

- **Contract**: Methodology configuration (tool registry, content tags, onboarding config, community config) is published and consumed by all domains.
- **Direction**: Methodology → (Learning, Marketplace, Social, Onboarding, Discovery, AI, Compliance, Notifications)
- **Trigger**: On methodology definition change (admin action) or family methodology selection change (user action).
- **Guarantee**: Domains MUST reflect methodology changes within a reasonable propagation window (SHOULD be < 1 minute for user-facing changes).

### 18.2 Learning → Compliance

- **Contract**: Learning data (activities, assessments, attendance, projects) flows to Compliance for report generation.
- **Direction**: Learning → Compliance
- **Data**: Aggregated activity records, assessment scores, attendance logs, and portfolio artifacts.
- **Privacy**: Data stays within the family's scope — no cross-family aggregation.

### 18.3 Learning → Social

- **Contract**: Learning milestones generate social events (timeline posts).
- **Direction**: Learning → Social
- **Data**: Milestone type, student name (if parent allows), subject, and achievement description.
- **Privacy**: Milestone sharing MUST be opt-in per milestone or configurable as a default.

### 18.4 Marketplace → Learning

- **Contract**: Purchased content metadata flows to Learning for tool integration.
- **Direction**: Marketplace → Learning
- **Data**: Content metadata (title, methodology, subjects, content type), purchase status, and file access references.
- **Trigger**: On purchase completion.

### 18.5 All → AI

- **Contract**: Anonymized, aggregated usage data flows to AI for recommendation training and generation.
- **Direction**: (Learning, Social, Marketplace) → AI
- **Data**: Activity patterns, content popularity signals, methodology correlations — all anonymized and aggregated.
- **Privacy**: MUST NOT include PII. MUST comply with §10.4 ethical requirements.

### 18.6 Discovery → Onboarding

- **Contract**: Quiz results transfer from Discovery (pre-account) to Onboarding (post-account).
- **Direction**: Discovery → Onboarding
- **Data**: Quiz result identifier, methodology scores, and recommendations.
- **Trigger**: User creates account and opts to import quiz results.

---

## 19. Phased Rollout & MVP Definition `[V§11]`

### Phase 1 — MVP

**Goal**: Launch a usable platform with core social, basic learning, marketplace access, and discovery tools.

| Domain | In Scope |
|--------|----------|
| **Identity & Access** | Family accounts, parent users, student profiles, email + OAuth authentication |
| **Methodology** | 6 methodologies (Charlotte Mason, Traditional, Classical, Waldorf, Montessori, Unschooling), tool registry, philosophy modules |
| **Discovery** | Methodology quiz, methodology explorer pages, state legal guides (all 51), Homeschooling 101, advocacy content |
| **Onboarding** | Full onboarding flow: account creation, family setup, methodology wizard, getting-started roadmaps, starter recommendations, community connections |
| **Social** | Profiles, timeline/feed (reverse-chronological), comments (threaded), friend system, direct messaging (friends-only), platform-managed methodology groups, events (basic), location-based discovery (opt-in) |
| **Learning** | Activities, Reading Lists, Journaling & Narration, basic Progress Tracking |
| **Marketplace** | Creator onboarding, content listings, browse/search/filter, purchase flow (cart + checkout), ratings & reviews (verified-purchaser), basic creator dashboard |
| **Trust & Safety** | CSAM detection (PhotoDNA/NCMEC), automated content screening, user reporting system, bot prevention (CAPTCHA + rate limiting), community guidelines, basic moderation dashboard |
| **Billing** | Free tier, marketplace transactions, payment processing, sales tax |
| **Notifications** | In-app notification center, transactional email (account, purchases, social) |
| **Search** | Full-text search (social + marketplace), basic autocomplete |
| **Content & Media** | Image/file upload, storage, delivery |
| **Privacy/Compliance** | COPPA compliance, privacy policy, terms of service, data export |

**Not in Phase 1**: Premium subscriptions, AI recommendations, compliance reporting, portfolios, transcripts, user-created groups, advanced analytics, methodology-specific advanced tools.

### Phase 2 — Premium & Depth

**Goal**: Introduce premium subscription, deepen learning tools, and expand social features.

| Addition | Details |
|----------|---------|
| **Premium subscription** | Family-level billing, monthly/annual, upgrade/downgrade flows |
| **Compliance reporting** | State-specific configuration, attendance logging, assessment records |
| **AI recommendations** | Methodology-constrained content and activity recommendations |
| **User-created groups** | Open, request-to-join, and invite-only groups |
| **Events (full)** | Recurring events, capacity management, group-linked events |
| **Tests & Grades** | Assessment recording, grading scales, running averages |
| **Projects** | Multi-step project tracking with milestones |
| **Advanced analytics** | Trend visualization, subject balance, progress reports |
| **Mentorship** | Methodology-matched mentor/mentee connections |
| **Creator payouts** | Payout scheduling, 1099-K, advanced creator analytics |

### Phase 3 — Scale & Specialize

**Goal**: Add methodology-specific tools, advanced compliance features, and begin scaling infrastructure.

| Addition | Details |
|----------|---------|
| **Methodology-specific tools** | Nature journals, trivium tracker, rhythm planner, observation logs, habit tracking, interest-led logs, handwork tracker, practical life activities |
| **Mastery paths** | Beginner → intermediate → advanced methodology education |
| **Advanced AI** | Improved recommendations, data collection for future tutoring |
| **Transcripts** | Formal high school transcript generation |
| **Portfolios** | Customizable PDF portfolio generation |
| **Mobile apps** | Native iOS and Android applications |
| **Enhanced moderation** | Grooming detection, advanced behavioral analysis |

### Phase 4 — Expansion

**Goal**: Expand platform capabilities and reach.

| Addition | Details |
|----------|---------|
| **Video conferencing** | Virtual co-op sessions `[V§11]` |
| **Tutoring marketplace** | Tutoring service listings and booking `[V§10]` |
| **Partnerships** | Curriculum publisher integrations `[V§10]` |
| **Internationalization** | Multi-language, multi-currency, international curriculum standards |
| **Advanced AI tutoring** | AI-powered tutoring features (using Phase 2-3 data) |

---

## 20. Open Questions

Each question includes a recommended position based on analysis of the vision document, competitive landscape, and common platform patterns. These are starting points for decision-making, not final answers.

### 20.1 Marketplace Revenue Share Percentage

**Question**: What percentage of each marketplace sale goes to the creator vs. the platform?

**Recommended Position**: **70-75% to creator**. This is competitive with Outschool (70%) and significantly better than Teachers Pay Teachers' base rate (55-60%). A creator-friendly split aligns with `[V§12]` ("Creator-friendly — fair revenue sharing") and is essential for attracting quality content in a new marketplace.

### 20.2 Premium Subscription Pricing

**Question**: What should the premium subscription cost?

**Recommended Position**: **~$10-15/month (family-level)**, with an annual discount of ~20%. This positions between homeschool planner pricing ($5-8/mo) and Schoolio's per-student model ($30/mo per student). Family-level pricing is a competitive differentiator per `[V§10]`.

### 20.3 Per-Student vs. Per-Family Methodology Selection

**Question**: Should methodology be selected at the family level or per-student?

**Recommended Position**: **Family-level primary/secondary with per-student overrides**. This reduces onboarding complexity for the majority case (family uses one approach) while accommodating families with children at different stages (e.g., one child doing Classical, another doing Montessori). Specified in §4.6.

### 20.4 Student-Facing Views

**Question**: Should students have their own views into the platform?

**Recommended Position**: **MVP is parent-only**. Post-MVP (Phase 2-3), add supervised student views for ages 10+. The parent-mediated model `[V§7]` is the priority; student views add complexity around COPPA compliance, content filtering, and access controls that are better addressed after the core platform stabilizes.

### 20.5 Content Moderation Operational Model

**Question**: How should content moderation be staffed and operated?

**Recommended Position**: **Automated first pass + community group moderators + small in-house team for escalation**. Automated screening handles volume (spam, CSAM detection); community moderators handle methodology group dynamics; a small professional team handles appeals, edge cases, and child safety escalations. This scales with community growth while keeping costs manageable at launch.

### 20.6 Offline Access

**Question**: Should the platform support offline use?

**Recommended Position**: **Not in MVP**. Consider service-worker-based caching for read-only access to existing learning data in Phase 3. Full offline support (with sync) introduces significant complexity and is deferred. Web-first responsive design `[V§11]` is the priority.

### 20.7 Third-Party Integrations

**Question**: Should the platform integrate with external services beyond payment processing?

**Recommended Position**: **None for MVP** beyond payment processing. In Phase 3, explore integrations with library systems (book data), standardized testing providers (score import), and popular calendar applications (event sync). Each integration adds maintenance burden and third-party dependency.

### 20.8 Accessibility for Students with Disabilities

**Question**: How should the platform support students with learning disabilities or special needs?

**Recommended Position**: **WCAG 2.1 AA baseline** (§17.6) ensures the platform is accessible. Specialized tools (e.g., IEP tracking, accommodation logging, dyslexia-friendly rendering) SHOULD be deferred to Phase 3, informed by user research with special-needs homeschooling families. The flexible, methodology-agnostic approach already supports individualized learning.

### 20.9 Worldview Tagging Governance

**Question**: How should worldview tags on marketplace content be managed?

**Recommended Position**: **Creator self-reported with community flagging for miscategorization**. Creators select applicable worldview tags when listing content `[V§12]`. Users MAY flag content they believe is miscategorized. Flagged items are reviewed by moderators. This balances creator autonomy with community trust, aligned with content neutrality `[V§12]`.

---

## 21. Glossary

| Term | Definition |
|------|-----------|
| **Activity** | A logged learning event — a lesson, reading session, field trip, or any educational activity recorded in the platform. |
| **Charlotte Mason** | An educational methodology emphasizing living books, narration, nature study, short lessons, and habit formation. Named after the 19th-century British educator. |
| **Classical Education** | An educational methodology structured around the trivium: grammar (foundational knowledge), logic (analytical thinking), and rhetoric (expression). Emphasizes great books, Latin, and Socratic discussion. |
| **COPPA** | Children's Online Privacy Protection Act — US federal law regulating the collection of personal information from children under 13. |
| **Creator** | An individual or organization that lists and sells educational content through the marketplace. |
| **CSAM** | Child Sexual Abuse Material — illegal content that the platform is legally and morally obligated to detect, remove, and report. |
| **Eclectic Homeschooling** | An approach that combines elements from multiple methodologies. On the platform, this is expressed by selecting multiple methodologies (primary + secondary), not as a separate methodology. |
| **Family Account** | The top-level entity grouping parents and students. All billing, subscriptions, and methodology selections are at the family level. |
| **Free Tier** | The subscription level available to all users at no cost, including social features, basic learning tools, and marketplace access. |
| **Getting-Started Roadmap** | A methodology-specific, age-adapted checklist of concrete first steps for new homeschooling families. |
| **Living Books** | A Charlotte Mason concept — books written by a single author with passion and literary quality, as opposed to textbooks. Used in Reading Lists and curriculum recommendations. |
| **Mastery Path** | A structured progression of parent education content for a methodology, from beginner through advanced concepts. |
| **Methodology** | A coherent educational philosophy and practice (e.g., Charlotte Mason, Classical, Waldorf) that shapes the platform experience — tools, content, recommendations, and community. |
| **Methodology Explorer** | Public-facing deep-dive pages for each methodology, accessible without an account. |
| **Methodology Quiz** | A public-facing interactive assessment that recommends methodologies based on family values and preferences. |
| **Montessori** | An educational methodology emphasizing prepared environments, self-directed activity, hands-on learning, and observation. Named after Dr. Maria Montessori. |
| **Narration** | A Charlotte Mason technique where a child retells in their own words what they have read or heard, used as both a learning tool and assessment method. |
| **NCMEC** | National Center for Missing & Exploited Children — the organization to which platforms must report CSAM under US law. |
| **Parent User** | An authenticated individual (parent or guardian) associated with a Family Account, responsible for managing student profiles and family settings. |
| **PhotoDNA** | A hash-based technology for detecting known CSAM images, developed by Microsoft and used by NCMEC. |
| **Premium Tier** | The paid subscription level adding advanced features: compliance reporting, AI recommendations, advanced analytics, and enhanced tools. |
| **Primary Methodology** | The main methodology selected by a family, which shapes the default dashboard, terminology, and UX feel. |
| **RFC 2119** | "Key words for use in RFCs to Indicate Requirement Levels" — defines MUST, SHOULD, and MAY as used in this document. |
| **Secondary Methodology** | Additional methodology(ies) selected by a family, which add their tools to the available set alongside the primary methodology. |
| **Student Profile** | A child's profile within a Family Account. Students do not have independent accounts — all activity is mediated through a parent. |
| **Tool Registry** | The mapping of learning tools to methodologies, determining which tools are available for each methodology selection. |
| **Traditional Homeschooling** | An educational methodology that mirrors conventional school structure: textbooks, worksheets, tests, grades, and structured schedules, but delivered at home. |
| **Trivium** | The three stages of Classical education: grammar (ages ~6-10), logic/dialectic (ages ~10-14), and rhetoric (ages ~14-18). |
| **Unschooling** | An educational methodology emphasizing child-led, interest-driven learning without formal curriculum or structure. |
| **Verified Purchaser** | A user who has purchased a marketplace listing and is therefore eligible to leave a rating or review. |
| **Waldorf Education** | An educational methodology emphasizing rhythm, artistic expression, imaginative play, and developmentally-appropriate learning. Named after the first Waldorf school founded by Rudolf Steiner. |
| **Worldview Tag** | A creator-applied label indicating the religious, secular, or philosophical perspective of marketplace content, enabling families to filter content to their preferences. |

---

*This specification covers the complete Homegrown Academy platform as defined in the product vision (VISION.md §1-§12). It is technology-agnostic and intended to be the bridge between vision and engineering design. All requirements use RFC 2119 language and trace back to the vision document. Open questions (§20) capture decisions that require stakeholder input before implementation.*
