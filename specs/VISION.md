# Homegrown Academy — Product Vision

## 1. Mission

Children in the United States spend roughly 14,000 hours — from age five to eighteen — inside institutions that separate them from their families for the majority of their waking lives. These institutions suppress natural curiosity in favor of compliance, enforce age-segregated conformity, and measure success by standardized test performance. Children are sorted, ranked, and processed through a system designed for administrative efficiency, not human flourishing.

Homeschooling restores what that system takes away. Family presence. Individualized learning. The freedom for a child to develop at their own pace and according to their own nature. A seven-year-old who reads voraciously isn't held back to grade level; a ten-year-old who needs more time with fractions isn't shamed for falling behind a bell curve. Parents know their children. Schools process them.

**Homegrown Academy exists to make homeschooling accessible to every family that wants it.** The mission succeeds when more families homeschool — whether or not they use the platform.

The platform is a tool in service of that mission. It removes barriers: the administrative complexity that overwhelms new families, the isolation that discourages them, the fragmented tooling that wastes their time. It builds the community infrastructure that homeschooling families deserve but have never had. And it meets families at the very beginning — before they've made the decision, before they know what methodology means, before they've pulled a child out of school — because that is where the mission starts.

---

## 2. Overview

### Project Name
Homegrown Academy

### Elevator Pitch
A hybrid social-media + learning subscription platform that helps homeschooling families track progress, discover methodology-aligned content, and build local & online community relationships.

### Why It Matters
- **Barriers prevent families from starting — or make them quit.** Homeschooling parents currently cobble together fragmented tools (planners, Facebook groups, curriculum marketplaces, record-keeping apps) with no integration between them. This administrative burden is one of the top reasons families never start homeschooling or return to conventional school within the first two years. Homegrown Academy unifies curriculum planning, progress tracking, and community coordination into a single platform — removing the friction that costs families their freedom.
- **Methodology empowerment**: Parents deserve tools that respect and support their chosen educational philosophy — not a one-size-fits-all curriculum. Homegrown Academy delivers methodology-specific functionality, curated content, and AI-driven recommendations tailored to how each family actually homeschools.
- **Sustainable business model**: Tiered family subscriptions and a revenue-share curriculum marketplace create diversified, recurring revenue while keeping the core social experience free to drive network effects. Revenue sustains the mission — it does not define it.

---

## 3. Market Context

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

## 4. Stakeholders & Personas

### End Users
- **Prospective homeschoolers**: Families exploring whether homeschooling is right for them. They interact with public-facing discovery tools — the methodology quiz, state legal guides, and educational content — before creating an account. This audience is critical because serving them IS the mission: every family that gains the confidence and information to start homeschooling is a success, regardless of whether they become platform users.
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

## 5. Core Architecture: Methodology as a First-Class Concept

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

## 6. Discovery & Onboarding

The platform does not assume families have already decided to homeschool or already know their methodology. Prospective and new homeschoolers feel overwhelmed — by the decision itself, by legal requirements, by the sheer number of approaches. The platform meets them where they are and guides them forward.

### Public-Facing Discovery (Pre-Account)

These tools are accessible without an account. They are genuine resources AND the mission in action — helping families make informed decisions and take their first steps.

- **"What methodology fits your family?" quiz** — An interactive assessment that asks about the family's values, learning preferences, and practical constraints, then recommends methodologies with clear explanations of why each is a fit. Not a lead-capture form disguised as a quiz — a genuinely useful tool that families share with each other.
- **Methodology explorer** — Deep-dive pages for each methodology: philosophy and history, what a typical day looks like, pros and cons, recommended starting resources, and real family stories. Detailed enough that a parent can confidently choose an approach.
- **"How to start homeschooling in [state]" guides** — State-specific legal requirements, notification processes, record-keeping rules, assessment obligations, and umbrella school options. Authoritative, regularly updated, and genuinely hard to find elsewhere. (High-value SEO target — these searches have strong intent and poor existing results.)
- **Homeschooling 101 content** — Addressing the real concerns that stop families: socialization, cost, time commitment, dual-income families, special needs, transitioning from public school mid-year, and what to tell skeptical relatives.
- **The case for homeschooling** — Not defensive, not apologetic. Clear-eyed content about what conventional schooling does and what families gain by choosing differently. The same confidence that lives in the Mission section, made accessible to families still deciding.

### Guided Onboarding (Post-Account)

Once a family creates an account, the platform walks them through setup with methodology-aware guidance — not a generic settings wizard.

- **Family profile setup** — Children, ages, grade levels, and any relevant context (special needs, prior schooling, family schedule)
- **Methodology selection wizard** — Builds on the quiz if the family already took it. Deeper questions, video and text introductions to each methodology, and the ability to explore before committing. Families can select multiple methodologies (eclectic approach) with clear guidance on how that works.
- **Getting started roadmap** — A methodology-specific first-week checklist. Charlotte Mason families get a different roadmap than Traditional families. Concrete, actionable, and designed to build early confidence.
- **Starter curriculum recommendations** — Curated and specific: "the 3 most popular Charlotte Mason starter packages for a 2nd grader," not an overwhelming catalog dump. Drawn from the marketplace with community ratings.
- **Community connections** — Methodology-matched groups, nearby homeschool families, and mentor matching with experienced homeschoolers who use the same approach.

### Ongoing Parental Growth

Homeschooling is a practice, not a one-time decision. Parents grow into their methodology over years, and the platform supports that growth.

- **Methodology mastery paths** — Beginner, intermediate, and advanced modules for each methodology. A first-year Charlotte Mason parent learns about short lessons and narration; a third-year parent dives into handicrafts and nature study integration.
- **Seasonal and milestone guidance** — Age-appropriate suggestions as children grow: "Your child is entering the grammar stage — here's what changes in Classical education," or "Transitioning from early elementary to middle school with Waldorf."
- **Community mentorship** — Connecting experienced families with newer ones, methodology-matched. Structured enough to be useful, informal enough to feel like community rather than a program.
- **"Homeschool confidence" signals** — Progress dashboards and milestone celebrations that remind parents they are doing this well. Homeschooling parents, especially in their first years, need reassurance that their children are thriving. The platform provides it with data and community, not empty affirmations.

---

## 7. Social Layer

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
**Everything is private by default.** No public profiles, no public posts, no public user-generated content. All social content is visible only to friends. This is a non-negotiable design principle — details about children will be shared on this platform, and privacy must be absolute.

Public-facing discovery content (methodology guides, state legal guides, the methodology quiz) is intentionally accessible without an account. This is educational and informational content — no user-generated content, no personal data, and no social features are public. The privacy-first principle applies to all user and family data without exception.

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

## 8. Learning Layer

The learning layer provides methodology-scoped **interactive learning tools** for planning, executing, tracking, and directly engaging with a family's homeschool education. Students don't just have their progress logged — they actively use the platform to take quizzes, watch video lessons, read content, and progress through structured lesson sequences. Parents plan and assign; students (ages 10+, supervised) interact with content directly on the platform.

### Methodology-Scoped Learning Tools
The available tools vary based on the family's selected methodology(ies). The platform provides both **tracking tools** (parent-operated logging) and **interactive tools** (student-facing engagement):
- **Activities** — Logging and tracking daily learning activities
- **Tests & grades** — Traditional assessment tools including **online quiz-taking with auto-scoring** (primarily Traditional, Classical methodologies)
- **Reading lists** — Curated and custom book lists with **in-platform reading for purchased content** (primarily Charlotte Mason, Classical)
- **Journaling** — Student journals, nature journals, narration records
- **Projects** — Multi-step project tracking and documentation
- **Video lessons** — **In-platform video player** with adaptive streaming, progress tracking, and completion logging (self-hosted HLS + YouTube/Vimeo embeds)
- **Lesson sequences** — **Structured content paths** (lesson → reading → quiz → video) that students progress through, with parent override controls
- **Content viewer** — **In-platform PDF/document viewer** for purchased marketplace content
- **Progress tracking & analytics** — Visualize student progress over time across subjects and skills

### Supervised Student Views
Students aged 10 and above can access a **simplified, supervised interface** configured by their parent. The student view shows assigned content — quizzes to take, videos to watch, readings to complete, sequences to follow — without access to social features, the marketplace, or messaging. Parents have full visibility and administrative control over the student experience. This is not an independent account — it operates entirely within the parent's account, maintaining COPPA compliance.

### Parent Education
Tool-adjacent tips and methodology-specific guidance integrated directly into the learning tools — for example, "how to use narration effectively" surfaced within the narration tool, or "choosing living books" guidance within the reading list feature. Deeper methodology education and parental growth content lives in the Discovery & Onboarding section (§6, Ongoing Parental Growth).

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

## 9. Curriculum Marketplace

An open marketplace where independent creators sell digital curriculum and educational content directly to homeschool families. Purchased content is **consumable in-platform** — families view documents, watch videos, take quizzes, and progress through lesson sequences without leaving the platform. Downloads remain available as a fallback.

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
- **Interactive quizzes** — Auto-scored assessments students take online
- **Lesson sequences** — Structured content paths combining readings, videos, quizzes, and activities

### Creator Authoring Tools
Creators build interactive content using platform-provided authoring tools:
- **Quiz builder** — Create question banks and assemble quizzes with multiple question types (multiple-choice, fill-in-the-blank, true/false, matching, ordering, short answer)
- **Sequence builder** — Arrange content items into structured lesson paths that students progress through

### Integration with Platform
The marketplace is not a standalone storefront — it is deeply integrated with the rest of the platform:
- Purchased content connects to the family's planning and tracking tools
- **Purchased interactive content is playable in-platform** — quizzes are taken, videos are watched, sequences are followed, all within the learning tools
- AI recommendations draw from the marketplace catalog
- Community reviews and ratings inform discovery
- Methodology tagging ensures content appears in the right context

---

## 10. Revenue Model

The revenue model is designed to keep the social layer free (maximizing network effects and adoption) while generating sustainable revenue through multiple streams.

The free tier and public-facing tools are not loss leaders — they are the mission. Revenue sustains the mission; it does not define it.

### Free Tier — Growth Engine
- Full social features (timeline, posts, profiles, chat, groups, events, discovery)
- Basic methodology-scoped learning tools
- Marketplace access (browse, purchase, rate)
- Methodology education modules
- Discovery tools, methodology explorer, getting started roadmaps, and onboarding flow

The free tier must be genuinely useful — not a crippled experience. Its purpose is to drive adoption, build community, and generate marketplace transaction revenue.

### Premium Subscription — Depth & Power
A family-level subscription (not per-student) for advanced platform features:
- State compliance reporting and transcript generation
- Advanced progress analytics and insights
- AI-powered content recommendations
- Enhanced storage for journals, projects, and portfolios
- Advanced methodology mastery paths and personalized growth recommendations
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
- Lead generation from public-facing content (quiz completions, state guide visits → account creation)

---

## 11. Platform & Technology

### Platform Strategy
- **Web-first** — Launch as a responsive web application
- **Mobile later** — Native mobile apps (iOS/Android) as a subsequent phase
- **Tech stack** — Open to recommendations based on project requirements

### Future Platform Features
- **Group video conferencing** — For virtual co-op sessions, enabling families to learn together remotely. Not required for v1 but a key future capability.
- **Native mobile apps** — iOS and Android apps for on-the-go access to social features, progress logging, and marketplace browsing.

---

## 12. Guiding Principles

1. **Mission first** — The platform exists to help more families homeschool. Every product decision is measured against this purpose. Revenue sustains the mission; growth serves the mission. If a feature helps families homeschool but doesn't help the business, we still build it.
2. **Meet families where they are** — The platform serves families at every stage: considering, beginning, practicing, and mastering. No one should feel they need to already know what they're doing to benefit from what the platform offers.
3. **Privacy first** — All user and family data is private by default. No public profiles, no public posts, no public user-generated content. Children's data is sacred. The intentional exception is educational and informational content (methodology guides, state legal guides, the quiz) — these are public resources that contain no user data.
4. **Methodology respect** — The platform does not favor any methodology over another. Every approach is equally supported with real, dedicated functionality.
5. **Content neutrality** — Both religious and secular content is welcome. Content is tagged by worldview so families can filter to their preferences. The platform itself is neutral.
6. **Parent authority** — Parents are the ultimate decision-makers for their children's education and social interactions on the platform. The platform provides tools and information; parents make choices.
7. **Community over content** — The social layer and community connections are the core value proposition and growth engine, not the content itself.
8. **Creator-friendly** — The marketplace must be attractive to creators through fair revenue sharing, good tooling, and access to an engaged audience.
9. **Simplicity over complexity** — Reduce the administrative burden of homeschooling; don't add to it.
