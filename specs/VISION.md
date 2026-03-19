# Homegrown Academy — Product Vision

## 1. Overview

### Project Name
Homegrown Academy

### Elevator Pitch
A hybrid social-media + learning subscription platform that helps homeschooling families track progress, discover methodology-aligned content, and build local & online community relationships.

### Why It Matters
- **Administrative burden**: Homeschooling parents currently cobble together fragmented tools (planners, Facebook groups, curriculum marketplaces, record-keeping apps) with no integration between them. Homegrown Academy unifies curriculum planning, progress tracking, and community coordination into a single platform.
- **Methodology empowerment**: Parents deserve tools that respect and support their chosen educational philosophy — not a one-size-fits-all curriculum. Homegrown Academy delivers methodology-specific functionality, curated content, and AI-driven recommendations tailored to how each family actually homeschools.
- **Sustainable business model**: Tiered family subscriptions and a revenue-share curriculum marketplace create diversified, recurring revenue while keeping the core social experience free to drive network effects.

---

## 2. Market Context

### Target Market
- **Geography**: United States (initial launch)
- **Audience**: Homeschooling families with children Pre-K through 12th grade
- **Market size**: $3.5B (2024), projected $7.2B by 2033 (8.5% CAGR)
- **Users**: 3.7M+ homeschooled students in the US, growing at 5.4% annually

### Competitive Landscape
The homeschool technology market is large, growing rapidly, and highly fragmented. No existing competitor combines social networking, a curriculum marketplace, methodology-scoped tools, and AI recommendations into a single platform.

#### Key Competitors

| Competitor | What They Do | Key Gap |
|---|---|---|
| **Homeschool Panda** | Planning + basic social | No marketplace, no AI, tiny community, basic social features |
| **Teachers Pay Teachers** | Largest curriculum marketplace | Not homeschool-native, no social, no tracking, no methodology tagging |
| **Outschool** | Live classes marketplace ($200M+ rev) | No planning, no community between parents, no ongoing curriculum |
| **Khan Academy** | Free K-12 content + Khanmigo AI tutor | No social, no marketplace, single pedagogy, not homeschool-specific |
| **LearnSpark** | AI-adaptive lesson plans | No community, no marketplace, new/unproven |
| **Schoolio** | Most "all-in-one" (curriculum + community) | No marketplace, no AI, per-student pricing, K-8 only, small user base |
| **My School Year / Homeschool Planet** | Record-keeping and planning | No social features, no marketplace, no AI, purely administrative tools |
| **Facebook Groups** | Where most homeschool community lives | Not purpose-built, no educational integration, privacy concerns, ad-supported |

#### Strategic Differentiators
1. **Methodology as architecture** — No competitor tailors the entire tool experience by educational methodology. This is the platform's most unique differentiator.
2. **Purpose-built social layer** — Homeschool families are scattered across Facebook Groups with no dedicated, privacy-first alternative. This is the widest-open gap in the market.
3. **Integrated marketplace** — Unlike TPT or Outschool, the marketplace is natively connected to planning tools, social recommendations, and methodology tagging.
4. **Network effects as moat** — The combination of social + marketplace creates compounding value that standalone tools cannot replicate. Each new family makes the community more valuable; each new creator makes the marketplace more valuable.

---

## 3. Stakeholders & Personas

### End Users
- **Parents (account owners)**: The primary users. They manage family accounts, select methodologies, plan curriculum, track progress, participate in the social layer, and purchase marketplace content.
- **Students (parent-mediated accounts)**: Children Pre-K through 12. They do not have independent accounts — all student activity is managed through and visible to the parent account. Students interact with age-appropriate learning tools curated by their parents.
- **Content/curriculum creators**: Independent educators, curriculum developers, and publishers who sell digital resources through the marketplace. They set prices, manage listings, and earn revenue through the platform's revenue-share model.

### Internal Stakeholders
- Product
- Engineering
- Curriculum Partnerships
- Trust & Safety / Moderation
- DevOps

---

## 4. Core Architecture: Methodology as a First-Class Concept

The platform's central organizing principle is the **educational methodology**. A methodology is not just a content tag — it is a first-class architectural entity that shapes the entire user experience.

### What a Methodology Includes
- **A curated set of tools and functionality** — Each methodology enables a specific collection of learning tools. For example:
  - *Charlotte Mason*: nature study journals, living books lists, narration tools, habit tracking
  - *Traditional*: textbook assignments, quizzes, grade tracking, worksheets
  - *Waldorf*: rhythm/routine planning, artistic expression tools, handwork project tracking
  - *Montessori*: prepared environment planning, observation logs, practical life activities
  - *Classical*: trivium stage tracking, Socratic discussion logs, Latin/logic tools
  - *Unschooling*: interest-led activity logging, documentation tools, portfolio building
- **Philosophy education modules** — Resources explaining the methodology's principles, history, and practical application
- **Parent resources** — Guides, tips, and community content specific to the methodology
- **Content tags** — All marketplace content is tagged to applicable methodologies (e.g., living books → Charlotte Mason, textbooks → Traditional/Classical)

### Methodology Selection
- Parents select one or more methodologies during onboarding
- A **primary methodology** shapes the default dashboard and overall UX feel
- **Secondary methodologies** add their tools to the available set
- This multi-methodology approach naturally supports **eclectic homeschoolers** (the most common approach) — selecting CM + Traditional unions both toolsets without needing an artificial "eclectic" category
- Families can change their methodology selection at any time

### Impact on User Experience
A Charlotte Mason parent logging in sees a fundamentally different platform than a Traditional or Waldorf parent — different tools, different content recommendations, different community suggestions. The methodology shapes everything.

---

## 5. Social Layer

The social layer is a full-featured, private social network purpose-built for homeschooling families. It is **free for all users** to maximize adoption and drive network effects.

### Core Social Features
- **Profiles** — Family profiles with parent and student information
- **Timeline / feed** — Posts, updates, and activity from friends
- **Comments** — On posts and shared content
- **Friend lists** — Connect with other homeschool families
- **Direct messaging** — Private conversations between parents
- **Groups** — Built-in groups per methodology + custom/community-created groups
- **Events** — Local event creation, discovery, and coordination
- **Location-based discovery** — Find nearby homeschool families, groups, and events

### Privacy Model
**Everything is private by default.** No public profiles, no public posts, no public content. All social content is visible only to friends. This is a non-negotiable design principle — details about children will be shared on this platform, and privacy must be absolute.

### Child Safety
- **Parent-mediated accounts only** — Children do not have independent accounts. All student activity is managed through the parent account.
- **Full parental visibility** — Parents have complete access to all communications involving their children.
- **Friendship approval** — Parents must approve any social connections for their children.
- **COPPA compliance** — Mandatory. The platform must comply with the Children's Online Privacy Protection Act given the Pre-K through 12 age range.

### Content Moderation
- **CSAM** — Instantly removed and reported to authorities. Zero tolerance. This is both a legal requirement and a moral imperative.
- **Child-facing content** — Parents are the primary moderators of their children's experience, supported by platform safety tools.
- **Parent-to-parent interactions** — Minimal but present platform moderation. The specific mechanism is to be determined, but the need is clear: inter-methodology hostility and personal attacks must be prevented from escalating. Fighting and hostility between groups (e.g., religious vs. secular, methodology debates) is actively discouraged.
- **Bot detection** — Robust automated detection and prevention of bot accounts.

---

## 6. Learning Layer

The learning layer provides methodology-scoped tools for planning, executing, and tracking a family's homeschool education.

### Methodology-Scoped Learning Tools
The available tools vary based on the family's selected methodology(ies). Examples include:
- **Activities** — Logging and tracking daily learning activities
- **Tests & grades** — Traditional assessment tools (primarily Traditional, Classical methodologies)
- **Reading lists** — Curated and custom book lists (primarily Charlotte Mason, Classical)
- **Journaling** — Student journals, nature journals, narration records
- **Projects** — Multi-step project tracking and documentation
- **Video lessons** — Integration with video-based curriculum content
- **Progress tracking & analytics** — Visualize student progress over time across subjects and skills

### Parent Education
- Methodology philosophy modules — helping parents understand and implement their chosen approach
- Practical guides and resources for each methodology
- Community-sourced tips and advice

### AI-Powered Recommendations
AI serves as an **important enhancement** to the platform, not the core differentiator:
- Content suggestions based on methodology, student progress, and community signals
- Curriculum recommendations from the marketplace aligned to the family's approach
- Potential future expansion into adaptive learning paths and AI tutoring

### State Compliance Reporting (Premium)
- Attendance logs, assessment records, and portfolio generation for state reviews
- Tailored to state-specific homeschool requirements
- Transcript generation for high school students
- This is a **premium feature** — a key driver for subscription upgrades

---

## 7. Curriculum Marketplace

An open marketplace where independent creators sell digital curriculum and educational content directly to homeschool families.

### Marketplace Model
- **Open listing** — Any creator can list content after basic onboarding. There is no heavy vetting gate.
- **Quality controlled by parents** — Verified-purchaser ratings (anonymous) are the primary quality signal. Only families who have purchased content can rate it. This ensures authentic, trustworthy reviews.
- **Revenue share** — The platform takes a percentage of each sale; creators keep the majority. The specific split is TBD (market benchmarks: Teachers Pay Teachers gives creators 55-80%; Outschool gives 70%).
- **Methodology tagging** — All content is tagged by applicable methodology, subject area, and age/grade range, enabling powerful discovery and filtering.

### Content Types
- Curriculum packages and lesson plans
- Worksheets and printable resources
- Unit studies and project guides
- Book lists and reading guides
- Video lessons and course content
- Assessment materials

### Integration with Platform
The marketplace is not a standalone storefront — it is deeply integrated with the rest of the platform:
- Purchased content connects to the family's planning and tracking tools
- AI recommendations draw from the marketplace catalog
- Community reviews and ratings inform discovery
- Methodology tagging ensures content appears in the right context

---

## 8. Revenue Model

The revenue model is designed to keep the social layer free (maximizing network effects and adoption) while generating sustainable revenue through multiple streams.

### Free Tier — Growth Engine
- Full social features (timeline, posts, profiles, chat, groups, events, discovery)
- Basic methodology-scoped learning tools
- Marketplace access (browse, purchase, rate)
- Methodology education modules

The free tier must be genuinely useful — not a crippled experience. Its purpose is to drive adoption, build community, and generate marketplace transaction revenue.

### Premium Subscription — Depth & Power
A family-level subscription (not per-student) for advanced platform features:
- State compliance reporting and transcript generation
- Advanced progress analytics and insights
- AI-powered content recommendations
- Enhanced storage for journals, projects, and portfolios
- Additional premium features TBD

Pricing TBD, but competitive positioning suggests a family-level price point (market benchmarks: planners are $5-8/mo; Schoolio is $30/mo per student).

### Marketplace Revenue Share — Broad Monetization
- Platform takes a percentage cut of every curriculum/content sale
- Monetizes all users (including free tier) — every marketplace purchase generates revenue
- Scales organically as the marketplace grows
- Creator-friendly split to attract quality content

### Future Revenue Opportunities
- Add-on services (professional transcript services, standardized test prep)
- Tutoring marketplace
- Sponsored/featured content placements (with clear labeling)
- Partnership integrations with curriculum publishers

---

## 9. Platform & Technology

### Platform Strategy
- **Web-first** — Launch as a responsive web application
- **Mobile later** — Native mobile apps (iOS/Android) as a subsequent phase
- **Tech stack** — Open to recommendations based on project requirements

### Future Platform Features
- **Group video conferencing** — For virtual co-op sessions, enabling families to learn together remotely. Not required for v1 but a key future capability.
- **Native mobile apps** — iOS and Android apps for on-the-go access to social features, progress logging, and marketplace browsing.

---

## 10. Guiding Principles

1. **Privacy first** — Everything is private by default. No public content. Children's data is sacred.
2. **Methodology respect** — The platform does not favor any methodology over another. Every approach is equally supported with real, dedicated functionality.
3. **Content neutrality** — Both religious and secular content is welcome. Content is tagged by worldview so families can filter to their preferences. The platform itself is neutral.
4. **Parent authority** — Parents are the ultimate decision-makers for their children's education and social interactions on the platform. The platform provides tools and information; parents make choices.
5. **Community over content** — The social layer and community connections are the core value proposition and growth engine, not the content itself.
6. **Creator-friendly** — The marketplace must be attractive to creators through fair revenue sharing, good tooling, and access to an engaged audience.
7. **Simplicity over complexity** — Reduce the administrative burden of homeschooling; don't add to it.
