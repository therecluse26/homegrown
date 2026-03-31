# Domain Spec 09 — Content & Media (media::)

## §1 Overview

The Content & Media domain is the **shared infrastructure service for all file uploads,
processing, storage, and delivery** across the platform. Every domain that handles user-generated
files — profile photos, journal images, marketplace content, nature study drawings, message
attachments — delegates upload orchestration and storage to `media::`. It owns the full upload
lifecycle from presigned URL generation through background processing (magic byte validation,
ffprobe analysis, CSAM scanning, content moderation, compression, variant generation) to
publication and CDN delivery. `[S§2.1, ARCH §8]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/media/` |
| **DB prefix** | `media_` `[ARCH §5.1]` |
| **Complexity class** | Simple — pipeline orchestration, no complex domain invariants `[ARCH §4.5]` |
| **CQRS** | No — read and write paths are straightforward; no separated query model needed |
| **External adapter** | `internal/media/adapters/s3.go` (S3-compatible object storage — provider-agnostic; Cloudflare R2 configured via endpoint URL) `[ARCH §2.10, §4.2]` |
| **Key constraint** | Magic byte validation required on all uploads `[CODING §5.2]`; every query family-scoped via `FamilyScope` `[CODING §2.4, §2.5]`; CSAM scan mandatory before publish `[S§12.1]` |

**What media:: owns**: Upload records and lifecycle state machine, presigned URL generation
(upload + download), magic byte validation, ffprobe metadata analysis, compression decision
logic and profile definitions, image variant generation (thumb + medium), storage key
management, S3-compatible storage adapter, orphan upload cleanup, context-based validation
rules (size limits, allowed types per context), processing job tracking, domain events
for upload state transitions.

**What media:: does NOT own**: CSAM detection adapters (owned by `safety::` —
`internal/safety/adapters/thorn.go`) `[ARCH §2.13]`, content moderation adapters (owned by
`safety::` — `internal/safety/adapters/rekognition.go`) `[ARCH §2.13]`, NCMEC reporting (owned by
`safety::service.ReportCSAM()`), social post/message attachment JSONB storage (owned by
`social::`) `[05-social §3.2]`, learning attachment JSONB fields (owned by `learn::`)
`[06-learn §3.2]`, marketplace file records (owned by `mkt::` — `mkt_listing_files`)
`[07-mkt §3.2]`, account/auth (owned by `iam::`), subscription tiers and storage quotas
(owned by `billing::` — Phase 3).

**What media:: delegates**: CSAM hash matching -> `safety.ThornAdapter` (Thorn Safer
PhotoDNA) `[ARCH §2.13]`. Content moderation -> `safety.RekognitionAdapter` (AWS
Rekognition label detection) `[ARCH §2.13]`. NCMEC reporting -> `safety.Service.ReportCSAM()`.
Background job scheduling -> hibiken/asynq `[ARCH §12]`. Asset compression -> FFMPEG worker
(stateless service, invoked via job queue).

---

## §2 Requirements Traceability

Every SPEC.md requirement that touches content or media maps to a section in this document.
Cross-references from other domain specs are included where `media::` is the provider.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Content & Media domain definition | `[S§2.1]` | §1 (overview) |
| Post attachments (photo post type requires images) | `[S§7.2.1]` | §9 (upload pipeline), §4.1 (request upload) |
| Message attachments (text and image) | `[S§7.5]` | §9 (upload pipeline), §4.1 (request upload) |
| Activity attachments (photos, files) | `[S§8.1.1]` | §9 (upload pipeline, `activity_attachment` context) |
| Journal entry image attachments | `[S§8.1.4]` | §9 (upload pipeline, `journal_image` context) |
| Project notes/attachments | `[S§8.1.5]` | §9 (upload pipeline, `project_attachment` context) |
| Nature journal drawing/photo support | `[S§8.1.8]` | §9 (upload pipeline, `nature_journal_image` context) |
| Marketplace listing files, preview content, thumbnail | `[S§9.2.1]` | §9 (upload pipeline, `marketplace_file`/`listing_preview`/`listing_thumbnail` contexts) |
| File versioning for purchasers | `[S§9.2.3]` | §16 (cross-domain — `mkt::` owns version tracking on `mkt_listing_files`) |
| CSAM detection on all uploaded images/videos | `[S§12.1]` | §10.3 (CSAM scan step), §16 (delegates to `safety::`) |
| Automated screening for explicit/adult content | `[S§12.2]` | §10.4 (content moderation step), §16 (delegates to `safety::`) |
| TLS encryption, file validation | `[S§17.1]` | §11 (magic byte validation), §9 (validation rules) |
| Performance targets | `[S§17.3]` | §9 (presigned URL response < 200ms), §10 (async processing) |

> **Coverage note on `[S§9.2.3]` (file versioning)**: SPEC.md §9.2.3 requires that purchasers
> receive updated files when creators publish new versions. This is implemented as follows:
> `mkt::` owns the `mkt_listing_files.version` column and the business logic for version
> increments. When a creator uploads a new version, `mkt::` calls `media.MediaService` to
> generate the presigned upload URL and manage the new upload. The old file's `storage_key`
> remains in `media_uploads` (soft-deleted); the new file gets a fresh `media_uploads` record.
> Media:: does not track versions — it manages individual uploads. Versioning is `mkt::`'s
> responsibility.

---

## §3 Database Schema

### §3.1 Enums

Implemented as `CHECK` constraints (not PostgreSQL ENUM types) per `[CODING §4.1]`:

```sql
-- Upload lifecycle status
-- CHECK: status IN ('pending', 'uploaded', 'processing', 'published',
--                   'quarantined', 'rejected', 'flagged', 'expired', 'deleted')

-- Upload context — determines validation rules (size limits, allowed types)
-- CHECK: context IN ('profile_photo', 'post_attachment', 'message_attachment',
--                    'activity_attachment', 'journal_image', 'nature_journal_image',
--                    'project_attachment', 'reading_cover', 'marketplace_file',
--                    'listing_preview', 'listing_thumbnail', 'creator_logo',
--                    'data_export', 'audio_attachment', 'video_lesson')
```

**Status state machine**:

```
pending ──► uploaded ──► processing ──► published
   │                        │
   │                        ├──► quarantined  (CSAM detected — immediate, irreversible)
   │                        │
   │                        ├──► rejected     (auto-rejected by content policy — e.g. nudity)
   │                        │
   │                        └──► flagged      (moderation concern — admin review required)
   │
   └──► expired  (presigned URL expired, upload never completed — orphan cleanup)

published ──► deleted  (soft-delete by owner, Phase 2)
flagged   ──► published  (admin override after review, Phase 2)
rejected  ──► published  (admin override via appeal, Phase 2)
```

### §3.2 Tables

> **Note**: The spec uses `uuidv7()` for PK defaults, but the actual migration uses
> `gen_random_uuid()` because `uuidv7()` requires a PostgreSQL extension (`pg_uuidv7`)
> that may not be available in all environments. Application code generates UUIDs via
> `uuid.NewV7()` (Go), so the DB default is only a fallback for direct SQL inserts.

```sql
-- ─── media_uploads ──────────────────────────────────────────────────────
-- Core upload tracking table. Every file that enters the system gets a row
-- here regardless of which domain initiated the upload. Family-scoped.
-- [S§2.1, ARCH §8.1]
CREATE TABLE media_uploads (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    uploaded_by           UUID NOT NULL REFERENCES iam_parents(id) ON DELETE CASCADE,

    -- Classification
    context               TEXT NOT NULL
                          CHECK (context IN (
                              'profile_photo', 'post_attachment', 'message_attachment',
                              'activity_attachment', 'journal_image', 'nature_journal_image',
                              'project_attachment', 'reading_cover', 'marketplace_file',
                              'listing_preview', 'listing_thumbnail', 'creator_logo',
                              'data_export', 'audio_attachment', 'video_lesson'
                          )),
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN (
                              'pending', 'uploaded', 'processing', 'published',
                              'quarantined', 'rejected', 'flagged', 'expired', 'deleted'
                          )),

    -- File metadata (provided at request time)
    original_filename     TEXT NOT NULL,
    content_type          TEXT NOT NULL,           -- validated MIME type

    -- File metadata (populated after upload confirmation)
    size_bytes            BIGINT,                  -- NULL until confirmed

    -- Storage
    storage_key           TEXT NOT NULL UNIQUE,     -- S3 object key

    -- Variants (set after processing)
    has_thumb             BOOLEAN NOT NULL DEFAULT false,
    has_medium            BOOLEAN NOT NULL DEFAULT false,

    -- Probe / compression metadata (populated by ProcessUploadJob)
    probe_metadata        JSONB,                   -- ffprobe analysis results (§10.5)
    original_size_bytes   BIGINT,                  -- pre-compression size, for analytics
    was_compressed        BOOLEAN NOT NULL DEFAULT false,

    -- Moderation (populated by safety:: scan results)
    moderation_labels     JSONB,                   -- Rekognition labels if flagged
    last_csam_scanned_at  TIMESTAMPTZ,             -- for periodic CSAM rescan [11-safety §10.7]

    -- Lifecycle
    expires_at            TIMESTAMPTZ,             -- presigned URL expiry (for orphan cleanup)
    published_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary query pattern: list uploads for a family filtered by context
CREATE INDEX idx_media_uploads_family_context
    ON media_uploads(family_id, context);

-- FK index on uploaded_by [CODING §4.4]
CREATE INDEX idx_media_uploads_uploaded_by
    ON media_uploads(uploaded_by);

-- Orphan cleanup: find pending uploads past expiry [§12]
CREATE INDEX idx_media_uploads_pending_expired
    ON media_uploads(expires_at)
    WHERE status = 'pending';

-- Processing queries: find uploads needing processing
CREATE INDEX idx_media_uploads_processing
    ON media_uploads(status)
    WHERE status = 'processing';

-- Published lookups (most common read path)
CREATE INDEX idx_media_uploads_published
    ON media_uploads(status)
    WHERE status = 'published';

-- CSAM rescan: find published uploads needing periodic rescan [11-safety §10.7]
CREATE INDEX idx_media_uploads_csam_rescan
    ON media_uploads(last_csam_scanned_at NULLS FIRST)
    WHERE status = 'published';

-- ─── media_processing_jobs ──────────────────────────────────────────────
-- Tracks background processing jobs for each upload. Supports retry logic
-- and failure diagnosis. Internal only — never exposed via API.
CREATE TABLE media_processing_jobs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id             UUID NOT NULL REFERENCES media_uploads(id),
    job_type              TEXT NOT NULL
                          CHECK (job_type IN (
                              'process_upload', 'compress_asset', 'cleanup_orphans',
                              'transcode_video'
                          )),
    status                TEXT NOT NULL DEFAULT 'queued'
                          CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    error_message         TEXT,                    -- internal only, NEVER exposed in API [CODING §2.2, §5.2]
    attempts              INTEGER NOT NULL DEFAULT 0,
    max_attempts          INTEGER NOT NULL DEFAULT 3,
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_media_processing_jobs_upload
    ON media_processing_jobs(upload_id);

CREATE INDEX idx_media_processing_jobs_queued
    ON media_processing_jobs(status)
    WHERE status IN ('queued', 'running');

-- ─── media_transcode_jobs ─────────────────────────────────────────────
-- Video transcoding pipeline for HLS adaptive bitrate streaming.
-- Creator uploads video → raw stored in R2 → transcode job generates
-- HLS playlist + segments at multiple quality levels. [S§8.1.11]
CREATE TABLE media_transcode_jobs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id             UUID NOT NULL REFERENCES media_uploads(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    input_key             TEXT NOT NULL,                  -- S3 key of raw uploaded video
    output_keys           JSONB,                          -- {master_playlist, variants: [{resolution, playlist_key, segment_prefix}]}
    resolutions           JSONB NOT NULL DEFAULT '["480p", "720p", "1080p"]',
    duration_seconds      INTEGER,                        -- detected from ffprobe
    error_message         TEXT,
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_media_transcode_jobs_upload ON media_transcode_jobs(upload_id);
CREATE INDEX idx_media_transcode_jobs_status ON media_transcode_jobs(status)
    WHERE status IN ('pending', 'processing');
```

### §3.3 RLS Policies

Family-scoped access on `media_uploads` via `family_id`. Enforced at the application layer
via `FamilyScope` `[CODING §2.4, §2.5, 00-core §8]`:

```sql
-- Application-layer enforcement: every query on media_uploads MUST include
-- WHERE family_id = $family_scope.family_id
-- Repository methods accept FamilyScope as first parameter to enforce this.
-- No direct table access without family_id filter.
```

`media_processing_jobs` is system-internal — accessed only by background jobs and admin
endpoints. No family-scoped RLS required (jobs reference `upload_id`, which is itself
family-scoped).

---

## §4 API Endpoints

**Prefix**: `/v1/media`

**Auth**: All endpoints require `AuthContext` (`FamilyScope`) `[00-core §7.2, §8]`.

### §4.1 Phase 1 Endpoints (3 endpoints)

#### POST /v1/media/uploads — Request Upload

Request a presigned upload URL for direct client -> S3 upload. `[ARCH §8.1]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `RequestUploadCommand`
```json
{
    "context": "journal_image",
    "content_type": "image/jpeg",
    "filename": "nature-walk.jpg",
    "size_bytes": 2048576
}
```
- **Response**: `201 Created` -> `UploadResponse`
```json
{
    "upload_id": "uuid",
    "presigned_url": "https://...",
    "storage_key": "uploads/{family_id}/{upload_id}/nature-walk.jpg",
    "expires_in_seconds": 3600
}
```
- **Validation**:
  - `content_type` must be in allowed types for the given `context` (§9.1)
  - `size_bytes` must be within the max size for the given `context` (§9.1)
  - `filename` is sanitized: stripped of path separators, control characters, truncated to 255 bytes
- **Side effects**:
  - Creates `media_uploads` record with status `pending`
  - Generates presigned PUT URL via `ObjectStorageAdapter.PresignedPut()`
  - Sets `expires_at` to `now() + 1 hour`
- **Error codes**:
  - `422` — invalid file type for context (`InvalidFileType`)
  - `422` — file size exceeds context limit (`FileTooLarge`)

#### POST /v1/media/uploads/:upload_id/confirm — Confirm Upload

Confirm that the client has completed the direct upload. Triggers background processing. `[ARCH §8.1]`

- **Auth**: Required (`FamilyScope`) — must be the upload owner
- **Response**: `200 OK` -> `UploadInfo`
```json
{
    "upload_id": "uuid",
    "status": "processing",
    "context": "journal_image",
    "content_type": "image/jpeg",
    "original_filename": "nature-walk.jpg",
    "size_bytes": 2048576,
    "urls": null,
    "has_thumb": false,
    "has_medium": false,
    "created_at": "2026-03-21T..."
}
```
- **Validation**:
  - Upload must exist and belong to the caller's family
  - Upload must be in `pending` status
  - Verifies the object exists in S3 via `ObjectStorageAdapter.GetObjectHead()`
  - Updates `size_bytes` from the actual object size (not the declared value)
- **Side effects**:
  - Transitions status: `pending` -> `uploaded` -> `processing`
  - Enqueues `ProcessUploadJob` on Default queue `[ARCH §12.2]`
  - Creates `media_processing_jobs` record with type `process_upload`
- **Error codes**:
  - `404` — upload not found or not owned by family (`UploadNotFound`)
  - `409` — upload not in `pending` status (`UploadNotConfirmed`)
  - `410` — presigned URL expired (`UploadExpired`)
  - `502` — S3 head check failed (`ObjectStorageError`)

#### GET /v1/media/uploads/:upload_id — Get Upload

Get upload status and URLs (original + variants). `[ARCH §8.1]`

- **Auth**: Required (`FamilyScope`)
- **Response**: `200 OK` -> `UploadInfo`
```json
{
    "upload_id": "uuid",
    "status": "published",
    "context": "journal_image",
    "content_type": "image/jpeg",
    "original_filename": "nature-walk.jpg",
    "size_bytes": 2048576,
    "urls": {
        "original": "https://media.homegrownacademy.com/uploads/{family_id}/{upload_id}/nature-walk.jpg",
        "thumb": "https://media.homegrownacademy.com/uploads/{family_id}/{upload_id}/nature-walk__thumb.jpg",
        "medium": "https://media.homegrownacademy.com/uploads/{family_id}/{upload_id}/nature-walk__medium.jpg"
    },
    "has_thumb": true,
    "has_medium": true,
    "created_at": "2026-03-21T...",
    "published_at": "2026-03-21T..."
}
```
- **URL generation rules**:
  - Published media: public CDN URL (`{OBJECT_STORAGE_PUBLIC_URL}/{storage_key}`)
  - Marketplace files: presigned GET URL (purchase verification required — never public)
  - Processing/pending media: `urls` is `null`
  - Quarantined/flagged media: `urls` is `null`, status reflects moderation state
- **Error codes**:
  - `404` — upload not found or not owned by family (`UploadNotFound`)

### §4.2 Phase 2 Endpoints (~3 endpoints)

#### DELETE /v1/media/uploads/:upload_id — Delete Upload

Soft-delete an upload. Sets status to `deleted`. Does not immediately remove the S3 object
(deferred to a cleanup job for safety).

- **Auth**: Required (`FamilyScope`) — must be the upload owner
- **Response**: `204 No Content`
- **Side effects**: Transitions status to `deleted`, sets `updated_at`
- **Error codes**: `404` (`UploadNotFound`), `403` (`NotOwner`)

#### GET /v1/media/uploads — List Uploads

List uploads for the authenticated family, with optional filtering.

- **Auth**: Required (`FamilyScope`)
- **Query**: `?context=journal_image&status=published&page=1&per_page=20`
- **Response**: `200 OK` -> `UploadListResponse` (paginated)

#### POST /v1/media/uploads/:upload_id/reprocess — Reprocess Upload (Admin)

Admin endpoint to re-trigger the processing pipeline for a specific upload. Used when
moderation results need re-evaluation or when the processing pipeline has been updated.

- **Auth**: Required (admin role)
- **Response**: `200 OK` -> `UploadInfo`
- **Side effects**: Re-enqueues `ProcessUploadJob`

---

## §5 Service Interface

`MediaService` is the **authoritative cross-domain interface** for all media operations.
Consumer domains (`social::`, `learn::`, `mkt::`) depend on this interface — not on domain-local
adapter sketches.

> **Reconciliation note**: `learn::MediaAdapter` (`06-learn §7`) and `mkt::MediaAdapter`
> (`07-mkt §7`) both sketch adapter interfaces for media operations. This spec defines the
> authoritative `MediaService` interface that supersedes both sketches. The existing adapter
> interface definitions in those specs should be understood as referring to `media.MediaService` —
> the implementation will inject a `MediaService` interface value where those specs reference
> `MediaAdapter`. The method signatures here are a superset of both sketches:
> `mkt.MediaAdapter.PresignedUpload()` maps to `RequestUpload()`;
> `mkt.MediaAdapter.PresignedGet()` maps to `PresignedGet()`;
> `learn.MediaAdapter.ValidateAttachment()` maps to `ValidateAttachment()`;
> `learn.MediaAdapter.GetUploadURL()` maps to `RequestUpload()`.

```go
// internal/media/ports.go

// MediaService is the authoritative media service interface consumed by all domains.
//
// Supersedes the MediaAdapter sketches in learn:: and mkt:: domain specs.
// Injected as a MediaService interface value into consumer domain services.
//
// All methods that access user data require family_id for family-scoping.
// [CODING §2.4]
type MediaService interface {

    // ─── Commands ───────────────────────────────────────────────────────

    // RequestUpload generates a presigned upload URL for direct client → S3 upload.
    //
    // Creates a media_uploads record in pending status, generates a
    // presigned PUT URL with context-appropriate size limits, and returns
    // the upload metadata. The client uploads directly to S3 — the server
    // never sees file bytes. [ARCH §8.1]
    RequestUpload(ctx context.Context, input *RequestUploadInput) (*UploadResponse, error)

    // ConfirmUpload confirms that the client has completed a direct upload.
    //
    // Verifies the object exists in S3, updates size_bytes from actual
    // object metadata, transitions status to processing, and enqueues
    // the ProcessUploadJob for background processing (magic bytes →
    // ffprobe → CSAM → moderation → compression → variants → publish).
    // [ARCH §8.1, §8.2]
    ConfirmUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error)

    // DeleteUpload soft-deletes an upload. (Phase 2)
    //
    // Transitions status to deleted. Does not immediately remove the
    // S3 object — deferred to a cleanup job.
    DeleteUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) error

    // ─── Queries ────────────────────────────────────────────────────────

    // GetUpload returns upload status and URLs (original + variants).
    //
    // Returns public CDN URLs for published media. Returns presigned GET
    // URLs for marketplace files (purchase verification is the caller's
    // responsibility — media:: generates the URL, mkt:: checks the
    // purchase). Returns null URLs for non-published statuses.
    GetUpload(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID) (*UploadInfo, error)

    // PresignedGet generates a presigned GET URL for secure file download.
    //
    // Used by mkt:: for purchased file downloads (1-hour expiry).
    // Callers are responsible for authorization checks (e.g., verifying
    // purchase exists) before calling this method. [ARCH §8.3]
    PresignedGet(ctx context.Context, storageKey string, expiresSeconds uint32) (string, error)

    // ValidateAttachment validates attachment metadata against context-based rules.
    //
    // Checks content_type and size_bytes against the allowed types and
    // max size for the given upload context. Does not touch S3 — this is
    // a pre-flight validation before generating a presigned URL.
    // Used by learn:: for attachment validation. [CODING §5.2]
    ValidateAttachment(ctx context.Context, uploadCtx UploadContext, contentType string, sizeBytes uint64) error
}
```

### §5.1 Service Implementation

```go
// MediaServiceImpl is the concrete implementation of MediaService.
//
// Orchestrates between repository, S3 adapter, safety adapters, and
// the background job system.
type MediaServiceImpl struct {
    uploads  UploadRepository
    jobs     ProcessingJobRepository
    storage  ObjectStorageAdapter
    safety   SafetyScanAdapter
    events   *EventBus
    config   *MediaConfig
}
```

**`MediaConfig`** holds runtime configuration:
```go
type MediaConfig struct {
    PublicURLBase            string // OBJECT_STORAGE_PUBLIC_URL env var
    PresignedUploadExpiry    uint32 // seconds, default 3600
    PresignedDownloadExpiry  uint32 // seconds, default 3600
}
```

---

## §6 Repository Interfaces

### §6.1 UploadRepository

```go
// UploadRepository defines persistence operations for media_uploads.
// All user-data queries are family-scoped via FamilyScope parameter. [CODING §2.4]
//
// Note: FamilyScope is passed by value (not pointer) to match the shared.FamilyScope
// convention used across all domains. Background-job methods omit it entirely.
type UploadRepository interface {

    // Create creates a new upload record in pending status.
    Create(ctx context.Context, scope FamilyScope, input *CreateUploadRow) (*Upload, error)

    // FindByID finds an upload by ID, scoped to family.
    FindByID(ctx context.Context, scope FamilyScope, uploadID uuid.UUID) (*Upload, error)

    // FindByIDUnscoped finds an upload by ID without family scope.
    // Used by background jobs (ProcessUploadJob, TranscodeVideoJob) that operate
    // outside of any user's auth context.
    FindByIDUnscoped(ctx context.Context, uploadID uuid.UUID) (*Upload, error)

    // UpdateStatus updates upload status and optional fields.
    UpdateStatus(ctx context.Context, uploadID uuid.UUID, status UploadStatus, updates *UploadStatusUpdate) (*Upload, error)

    // UpdateProbeMetadata updates probe metadata and compression info after processing.
    UpdateProbeMetadata(ctx context.Context, uploadID uuid.UUID, probe json.RawMessage, wasCompressed bool, originalSizeBytes *int64) error

    // SetVariantFlags sets variant flags after image processing.
    SetVariantFlags(ctx context.Context, uploadID uuid.UUID, hasThumb bool, hasMedium bool) error

    // SetModerationLabels sets moderation labels (from Rekognition results).
    SetModerationLabels(ctx context.Context, uploadID uuid.UUID, labels json.RawMessage) error

    // SetCSAMScannedAt records the timestamp of the last successful CSAM scan.
    // Called after ScanCSAM returns (whether clean or not) to maintain audit trail.
    SetCSAMScannedAt(ctx context.Context, uploadID uuid.UUID) error

    // List lists uploads for a family, filtered by context and/or status. (Phase 2)
    List(ctx context.Context, scope FamilyScope, filter *UploadListFilter, pagination *Pagination) (*PaginatedResult[Upload], error)

    // FindExpiredPending finds expired pending uploads for orphan cleanup.
    // System-internal — not family-scoped (runs as background job).
    FindExpiredPending(ctx context.Context, before time.Time, limit uint32) ([]Upload, error)
}
```

### §6.2 ProcessingJobRepository

```go
// ProcessingJobRepository defines persistence operations for media_processing_jobs.
// System-internal — no family-scoping needed (accessed only by background jobs).
type ProcessingJobRepository interface {

    // Create creates a new processing job record.
    Create(ctx context.Context, uploadID uuid.UUID, jobType string) (*ProcessingJob, error)

    // MarkRunning marks a job as running.
    MarkRunning(ctx context.Context, jobID uuid.UUID) error

    // MarkCompleted marks a job as completed.
    MarkCompleted(ctx context.Context, jobID uuid.UUID) error

    // MarkFailed marks a job as failed with an error message.
    MarkFailed(ctx context.Context, jobID uuid.UUID, errorMessage string) error

    // FindRetryable finds jobs eligible for retry (failed, attempts < max_attempts).
    FindRetryable(ctx context.Context, limit uint32) ([]ProcessingJob, error)
}
```

### §6.3 TranscodeJobRepository

```go
// TranscodeJobRepository defines persistence operations for media_transcode_jobs.
// System-internal — accessed only by the TranscodeVideoJob background worker.
type TranscodeJobRepository interface {

    // Create creates a new transcode job record.
    Create(ctx context.Context, uploadID uuid.UUID, inputKey string) (*TranscodeJob, error)

    // MarkRunning marks a transcode job as running.
    MarkRunning(ctx context.Context, jobID uuid.UUID) error

    // MarkCompleted marks a transcode job as completed with output keys and duration.
    MarkCompleted(ctx context.Context, jobID uuid.UUID, outputKeys json.RawMessage, durationSeconds int) error

    // MarkFailed marks a transcode job as failed with an error message.
    MarkFailed(ctx context.Context, jobID uuid.UUID, errorMessage string) error
}
```

---

## §7 Adapter Interfaces

### §7.1 ObjectStorageAdapter

**Provider-agnostic S3-compatible interface.** Uses standard S3 API operations (PutObject,
GetObject, HeadObject, DeleteObject, presigned URLs). Implemented via `aws-sdk-go-v2`
with custom endpoint URL — works with Cloudflare R2, AWS S3, MinIO, Backblaze B2, or any
S3-compatible provider. Provider is selected purely by configuration (endpoint URL, region,
credentials) — **zero code changes to switch providers**. `[ARCH §2.10, §4.2]`

Adapter file: `internal/media/adapters/s3.go` (named for the **protocol**, not the provider).

> **Note on naming**: `ARCHITECTURE.md §4.2` lists `internal/media/adapters/r2.go`. This spec
> renames it to `s3.go` to accurately reflect the provider-agnostic intent. The adapter
> uses the S3 protocol, not any R2-specific API. The ARCHITECTURE.md reference should be
> updated accordingly.

```go
// ObjectStorageAdapter defines the S3-compatible object storage interface.
//
// All operations use standard S3 API calls. The underlying provider
// (R2, S3, MinIO, etc.) is selected via endpoint URL configuration.
// [ARCH §2.10]
type ObjectStorageAdapter interface {

    // PresignedPut generates a presigned PUT URL for direct client upload.
    //
    // The URL includes Content-Type and Content-Length constraints
    // enforced by S3 at upload time.
    PresignedPut(ctx context.Context, key string, maxSizeBytes uint64, contentType string, expiresSeconds uint32) (string, error)

    // PresignedGet generates a presigned GET URL for secure file download.
    PresignedGet(ctx context.Context, key string, expiresSeconds uint32) (string, error)

    // PutObject uploads data from the server (for generated variants).
    //
    // Used when the server generates thumbnails/medium variants and
    // needs to write them to S3. NOT used for user uploads (those go
    // through presigned PUT URLs).
    PutObject(ctx context.Context, key string, data []byte, contentType string) error

    // GetObjectHead returns object metadata (size, content type) without downloading.
    //
    // Used during confirm_upload to verify the object exists and
    // record its actual size.
    GetObjectHead(ctx context.Context, key string) (*ObjectMetadata, error)

    // GetObjectBytes reads a byte range from an object.
    //
    // Used for magic byte validation — reads the first 16 bytes of
    // an uploaded file to verify the file type matches the declared
    // content_type. [CODING §5.2]
    GetObjectBytes(ctx context.Context, key string, start uint64, end uint64) ([]byte, error)

    // DeleteObject deletes an object from storage.
    DeleteObject(ctx context.Context, key string) error

    // DownloadToFile downloads an S3 object to a local file path.
    // Used by ffprobe/ffmpeg which operate on local files, not byte streams.
    DownloadToFile(ctx context.Context, key string, filepath string) error

    // UploadFromFile uploads a local file to S3.
    // Used after ffmpeg compression/transcoding writes output to a temp file.
    UploadFromFile(ctx context.Context, key string, filepath string, contentType string) error
}
```

**Configuration** (environment variables):

| Variable | Purpose | Example |
|----------|---------|---------|
| `OBJECT_STORAGE_ENDPOINT` | S3-compatible endpoint URL | `https://<account>.r2.cloudflarestorage.com` |
| `OBJECT_STORAGE_REGION` | Region | `auto` (R2), `us-east-1` (S3) |
| `OBJECT_STORAGE_BUCKET` | Bucket name | `homegrown-media` |
| `OBJECT_STORAGE_ACCESS_KEY_ID` | Credentials | (secret) |
| `OBJECT_STORAGE_SECRET_ACCESS_KEY` | Credentials | (secret) |
| `OBJECT_STORAGE_PUBLIC_URL` | CDN/public base URL for published media | `https://media.homegrownacademy.com` |

**Implementation sketch** (`internal/media/adapters/s3.go`):
```go
import (
    "context"
    "os"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3StorageAdapter struct {
    client    *s3.Client
    presigner *s3.PresignClient
    bucket    string
    publicURL string
}

func NewS3StorageAdapter(ctx context.Context) (*S3StorageAdapter, error) {
    // Build S3 client with custom endpoint from OBJECT_STORAGE_ENDPOINT
    // This single client works with R2, S3, MinIO, etc.
    endpoint := os.Getenv("OBJECT_STORAGE_ENDPOINT")
    region := os.Getenv("OBJECT_STORAGE_REGION")

    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(region),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
            os.Getenv("OBJECT_STORAGE_ACCESS_KEY_ID"),
            os.Getenv("OBJECT_STORAGE_SECRET_ACCESS_KEY"),
            "",
        )),
    )
    if err != nil {
        return nil, fmt.Errorf("loading AWS config: %w", err)
    }

    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.BaseEndpoint = &endpoint
    })

    return &S3StorageAdapter{
        client:    client,
        presigner: s3.NewPresignClient(client),
        bucket:    os.Getenv("OBJECT_STORAGE_BUCKET"),
        publicURL: os.Getenv("OBJECT_STORAGE_PUBLIC_URL"),
    }, nil
}
```

### §7.2 SafetyScanAdapter

Thin wrapper calling `safety::` domain services. Defined in `media::` as an inbound port —
implemented by `safety::` or a bridge adapter. `[ARCH §2.13]`

```go
// SafetyScanAdapter defines the safety scanning interface — delegates to safety:: domain.
//
// media:: calls these methods during ProcessUploadJob. The actual
// scanning implementations (Thorn Safer, AWS Rekognition) are owned
// by safety:: [ARCH §2.13].
type SafetyScanAdapter interface {

    // ScanCSAM scans for CSAM using Thorn Safer (PhotoDNA hash matching).
    //
    // Returns scan result. If CSAM is detected, the caller MUST
    // quarantine the upload immediately and invoke ReportCSAM().
    // [S§12.1]
    ScanCSAM(ctx context.Context, storageKey string) (*CSAMScanResult, error)

    // ScanModeration scans for content moderation violations using Rekognition.
    //
    // Returns moderation labels (explicit, violence, etc.). If
    // violations are detected, the caller flags the upload and
    // stores the labels. [S§12.2]
    ScanModeration(ctx context.Context, storageKey string) (*ModerationResult, error)

    // ReportCSAM reports confirmed/suspected CSAM to NCMEC.
    //
    // Called when ScanCSAM returns a positive result. Delegates to
    // safety.Service.ReportCSAM() which handles NCMEC filing,
    // evidence preservation, and account suspension.
    // [S§12.1, 18 U.S.C. § 2258A]
    ReportCSAM(ctx context.Context, uploadID uuid.UUID, scanResult *CSAMScanResult) error
}
```

---

## §8 Models (DTOs)

### §8.1 Request Types

```go
// internal/media/models.go

// RequestUploadCommand is the input for requesting a new upload.
// Used by the HTTP handler and by other domains calling MediaService.RequestUpload().
type RequestUploadCommand struct {
    Context     UploadContext `json:"context" validate:"required"`
    ContentType string       `json:"content_type" validate:"required,min=1,max=255"`
    Filename    string       `json:"filename" validate:"required,min=1,max=255"`
    SizeBytes   uint64       `json:"size_bytes" validate:"required"`
}

// RequestUploadInput is the internal input struct passed to MediaService.RequestUpload().
// Includes auth context fields not present in the HTTP body.
type RequestUploadInput struct {
    FamilyID    uuid.UUID
    UploadedBy  uuid.UUID
    Context     UploadContext
    ContentType string
    Filename    string
    SizeBytes   uint64
}

// ConfirmUploadCommand represents the confirm upload command (no body — upload_id from path, family from auth).
type ConfirmUploadCommand struct {
    UploadID uuid.UUID
    FamilyID uuid.UUID
}
```

### §8.2 Response Types

```go
// UploadResponse is the response from RequestUpload — contains presigned URL for direct upload.
type UploadResponse struct {
    UploadID         uuid.UUID `json:"upload_id"`
    PresignedURL     string    `json:"presigned_url"`
    StorageKey       string    `json:"storage_key"`
    ExpiresInSeconds uint32    `json:"expires_in_seconds"`
}

// UploadInfo contains upload information — returned by confirm and get endpoints.
type UploadInfo struct {
    UploadID         uuid.UUID   `json:"upload_id"`
    Status           string      `json:"status"`
    Context          string      `json:"context"`
    ContentType      string      `json:"content_type"`
    OriginalFilename string      `json:"original_filename"`
    SizeBytes        *int64      `json:"size_bytes"`
    URLs             *UploadURLs `json:"urls"`
    HasThumb         bool        `json:"has_thumb"`
    HasMedium        bool        `json:"has_medium"`
    CreatedAt        time.Time   `json:"created_at"`
    PublishedAt      *time.Time  `json:"published_at"`
}

// UploadURLs contains URLs for accessing the upload and its variants.
type UploadURLs struct {
    Original string  `json:"original"`
    Thumb    *string `json:"thumb"`
    Medium   *string `json:"medium"`
}

// UploadListResponse is a paginated list of uploads. (Phase 2)
type UploadListResponse struct {
    Uploads []UploadInfo `json:"uploads"`
    Total   uint64       `json:"total"`
    Page    uint32       `json:"page"`
    PerPage uint32       `json:"per_page"`
}
```

### §8.3 Internal Types

```go
// UploadContext determines validation rules (max size, allowed types).
// Maps to the CHECK constraint on media_uploads.context.
type UploadContext string

const (
    UploadContextProfilePhoto       UploadContext = "profile_photo"
    UploadContextPostAttachment     UploadContext = "post_attachment"
    UploadContextMessageAttachment  UploadContext = "message_attachment"
    UploadContextActivityAttachment UploadContext = "activity_attachment"
    UploadContextJournalImage       UploadContext = "journal_image"
    UploadContextNatureJournalImage UploadContext = "nature_journal_image"
    UploadContextProjectAttachment  UploadContext = "project_attachment"
    UploadContextReadingCover       UploadContext = "reading_cover"
    UploadContextMarketplaceFile    UploadContext = "marketplace_file"
    UploadContextListingPreview     UploadContext = "listing_preview"
    UploadContextListingThumbnail   UploadContext = "listing_thumbnail"
    UploadContextCreatorLogo        UploadContext = "creator_logo"
    UploadContextDataExport         UploadContext = "data_export"
    UploadContextAudioAttachment    UploadContext = "audio_attachment"
    UploadContextVideoLesson        UploadContext = "video_lesson"
)

// UploadStatus maps to CHECK constraint on media_uploads.status.
type UploadStatus string

const (
    UploadStatusPending     UploadStatus = "pending"
    UploadStatusUploaded    UploadStatus = "uploaded"
    UploadStatusProcessing  UploadStatus = "processing"
    UploadStatusPublished   UploadStatus = "published"
    UploadStatusRejected    UploadStatus = "rejected"
    UploadStatusQuarantined UploadStatus = "quarantined"
    UploadStatusFlagged     UploadStatus = "flagged"
    UploadStatusExpired     UploadStatus = "expired"
    UploadStatusDeleted     UploadStatus = "deleted"
)

// ImageVariant identifies image variant types.
type ImageVariant int

const (
    // ImageVariantThumb is 200x200 max, fit-within.
    ImageVariantThumb ImageVariant = iota
    // ImageVariantMedium is 800x800 max, fit-within.
    ImageVariantMedium
)

// CSAMScanResult is the CSAM scan result from Thorn Safer.
type CSAMScanResult struct {
    IsCSAM          bool
    Hash            *string  // PhotoDNA hash
    Confidence      *float64
    MatchedDatabase *string
}

// ModerationResult is the content moderation result from Rekognition.
//
// AutoReject and Priority are set by SafetyScanBridge (safety:: §11.2.2)
// based on the platform's label routing table. The RekognitionAdapter returns
// raw labels; the bridge applies routing decisions.
type ModerationResult struct {
    HasViolations bool
    AutoReject    bool               // true → status = rejected (not flagged)
    Labels        []ModerationLabel
    Priority      *string            // nil for auto-reject, "critical"|"high"|"normal" for flagged
}

// ModerationLabel represents a single moderation label from Rekognition.
type ModerationLabel struct {
    Name       string  `json:"name"`
    Confidence float64 `json:"confidence"`
    ParentName *string `json:"parent_name"`
}

// ObjectMetadata is returned by HEAD request.
type ObjectMetadata struct {
    ContentLength uint64
    ContentType   *string
}

// StorageError represents adapter error types.
type StorageError struct {
    Code    string
    Message string
}

func (e *StorageError) Error() string { return e.Message }

var (
    ErrObjectNotFound     = &StorageError{Code: "not_found", Message: "object not found"}
    ErrOperationFailed    = &StorageError{Code: "operation_failed", Message: "S3 operation failed"}
    ErrPresignFailed      = &StorageError{Code: "presign_failed", Message: "presigned URL generation failed"}
)

// ScanError represents safety scan error types.
type ScanError struct {
    Code    string
    Message string
}

func (e *ScanError) Error() string { return e.Message }

var (
    ErrScanUnavailable = &ScanError{Code: "unavailable", Message: "scan service unavailable"}
    ErrScanFailed      = &ScanError{Code: "failed", Message: "scan failed"}
)
```

---

## §9 Upload Pipeline (Deep-Dive 1)

The presigned URL upload pattern ensures the API server **never handles file bytes** —
the client uploads directly to S3-compatible storage. This keeps the server stateless,
avoids memory pressure from large files, and leverages the storage provider's global edge
network for fast uploads. `[ARCH §8.1]`

### §9.1 Context-Based Validation Rules

Each upload context has specific size limits and allowed content types:

| Context | Max Size | Allowed Types |
|---------|----------|--------------|
| `profile_photo` | 5 MB | `image/jpeg`, `image/png`, `image/webp` |
| `post_attachment` | 25 MB | `image/jpeg`, `image/png`, `image/webp`, `image/gif`, `video/mp4`, `video/webm` |
| `message_attachment` | 10 MB | `image/jpeg`, `image/png`, `image/webp`, `image/gif` |
| `activity_attachment` | 25 MB | `image/jpeg`, `image/png`, `image/webp`, `application/pdf` |
| `journal_image` | 10 MB | `image/jpeg`, `image/png`, `image/webp`, `image/gif` |
| `nature_journal_image` | 10 MB | `image/jpeg`, `image/png`, `image/webp` |
| `project_attachment` | 50 MB | `image/jpeg`, `image/png`, `image/webp`, `application/pdf`, `video/mp4` |
| `reading_cover` | 5 MB | `image/jpeg`, `image/png`, `image/webp` |
| `marketplace_file` | 500 MB | `application/pdf`, `application/zip`, `video/mp4`, `audio/mpeg` |
| `listing_preview` | 25 MB | `image/jpeg`, `image/png`, `image/webp`, `video/mp4` |
| `listing_thumbnail` | 5 MB | `image/jpeg`, `image/png`, `image/webp` |
| `creator_logo` | 2 MB | `image/jpeg`, `image/png`, `image/webp` |
| `data_export` | 5 GB | `application/zip` |
| `audio_attachment` | 50 MB | `audio/mpeg`, `audio/mp4`, `audio/wav`, `audio/flac`, `audio/aiff` |
| `video_lesson` | 5 GB | `video/mp4`, `video/quicktime`, `video/x-msvideo`, `video/webm` |

### §9.2 Storage Key Format

```
uploads/{family_id}/{upload_id}/{sanitized_filename}
```

- `family_id` ensures physical namespace isolation between families
- `upload_id` ensures uniqueness even for same-named files
- `sanitized_filename` preserves the original filename for user reference (stripped of path
  separators and control characters, truncated to 255 bytes)

### §9.3 Presigned URL Generation

```go
func (s *MediaServiceImpl) RequestUpload(ctx context.Context, input *RequestUploadInput) (*UploadResponse, error) {
    // 1. Validate context rules
    rules := getContextRules(input.Context)
    if !rules.AllowsContentType(input.ContentType) {
        return nil, ErrInvalidFileType
    }
    if input.SizeBytes > rules.MaxSizeBytes {
        return nil, ErrFileTooLarge
    }

    // 2. Generate storage key
    uploadID := uuid.NewV7()
    sanitized := sanitizeFilename(input.Filename)
    storageKey := fmt.Sprintf("uploads/%s/%s/%s", input.FamilyID, uploadID, sanitized)

    // 3. Generate presigned PUT URL
    presignedURL, err := s.storage.PresignedPut(
        ctx,
        storageKey,
        rules.MaxSizeBytes,
        input.ContentType,
        s.config.PresignedUploadExpiry,
    )
    if err != nil {
        return nil, fmt.Errorf("generating presigned URL: %w", err)
    }

    // 4. Create upload record
    upload, err := s.uploads.Create(ctx, &FamilyScope{FamilyID: input.FamilyID}, &CreateUploadRow{
        ID:               uploadID,
        FamilyID:         input.FamilyID,
        UploadedBy:       input.UploadedBy,
        Context:          input.Context,
        ContentType:      input.ContentType,
        OriginalFilename: sanitized,
        StorageKey:       storageKey,
        ExpiresAt:        time.Now().Add(time.Duration(s.config.PresignedUploadExpiry) * time.Second),
    })
    if err != nil {
        return nil, fmt.Errorf("creating upload record: %w", err)
    }

    return &UploadResponse{
        UploadID:         upload.ID,
        PresignedURL:     presignedURL,
        StorageKey:       storageKey,
        ExpiresInSeconds: s.config.PresignedUploadExpiry,
    }, nil
}
```

### §9.4 Upload Sequence

```
  Client                MediaService           S3 Storage              Job Queue
  │                        │                        │                       │
  │  1. POST /v1/media/    │                        │                       │
  │     uploads            │                        │                       │
  │  ─────────────────►    │                        │                       │
  │                        │  2. Validate context    │                       │
  │                        │     rules               │                       │
  │                        │  3. Create upload record │                       │
  │                        │  4. Generate presigned │                       │
  │                        │     PUT URL            │                       │
  │                        │  ────────────────────► │                       │
  │                        │                        │                       │
  │  5. Return upload_id   │                        │                       │
  │     + presigned URL    │                        │                       │
  │  ◄─────────────────    │                        │                       │
  │                        │                        │                       │
  │  6. PUT file bytes     │                        │                       │
  │     directly to S3     │                        │                       │
  │  ──────────────────────────────────────────────►│                       │
  │                        │                        │                       │
  │  7. POST /v1/media/    │                        │                       │
  │     uploads/:id/       │                        │                       │
  │     confirm            │                        │                       │
  │  ─────────────────►    │                        │                       │
  │                        │  8. HEAD object        │                       │
  │                        │     (verify exists,    │                       │
  │                        │      get actual size)  │                       │
  │                        │  ────────────────────► │                       │
  │                        │                        │                       │
  │                        │  9. Update status →    │                       │
  │                        │     processing         │                       │
  │                        │                        │                       │
  │                        │  10. Enqueue           │                       │
  │                        │      ProcessUploadJob  │                       │
  │                        │  ─────────────────────────────────────────────►│
  │                        │                        │                       │
  │  11. Return upload     │                        │                       │
  │      info (processing) │                        │                       │
  │  ◄─────────────────    │                        │                       │
  │                        │                        │   [Processing §10]    │
  │                        │                        │                       │
```

### §9.5 Orphan Detection

Uploads that remain in `pending` status after their `expires_at` timestamp are orphans —
the client requested an upload URL but never completed the upload (or never confirmed it).
These are cleaned up by `CleanupOrphanUploadsJob` (§12).

---

## §10 Asset Processing Pipeline (Deep-Dive 2)

After upload confirmation, the `ProcessUploadJob` orchestrates a unified processing pipeline
for **all asset types** (images, video, audio, documents). This extends the pipeline sketched
in `[ARCH §8.2]` with ffprobe analysis, compression decisions, and the FFMPEG worker
architecture.

> **Refinement of ARCH §8.2**: The architecture sketch shows content moderation running
> **after** thumbnail generation. This spec reorders the pipeline to run CSAM scanning and
> content moderation **before** compression and variant generation. Rationale: there is no
> value in generating thumbnails or compressing files that will be quarantined or flagged —
> early short-circuiting saves processing time and avoids creating variants of problematic
> content.

> **Refinement of ARCH §12.2**: The architecture lists two separate jobs — `ProcessImageJob`
> (Default queue) and `CsamScanJob` (Default queue). This spec consolidates them into a
> single `ProcessUploadJob` that handles the entire pipeline (magic bytes -> ffprobe -> CSAM ->
> moderation -> compression -> variants -> publish). Rationale: CSAM scanning is one step in a
> linear pipeline, not an independent job. A single orchestrator job is simpler to reason
> about, ensures correct ordering, and avoids the coordination overhead of chaining two
> separate jobs. Additionally, `ProcessUploadJob` handles all asset types (not just images),
> reflecting the broader scope of the media domain.

### §10.1 Pipeline Stages

Executed by `ProcessUploadJob` (Default queue):

```
ProcessUploadJob
    │
    ├─ 1. Magic byte validation (§11)
    │      Read first 16 bytes from S3
    │      Match against known file signatures
    │      FAIL → reject with InvalidFileType
    │
    ├─ 2. ffprobe analysis (inline, ~50-200ms)
    │      Extract: codec, bitrate, resolution, duration, format
    │      Store results as probe_metadata JSONB
    │      (ffprobe reads container headers only — safe for large files)
    │
    ├─ 3. CSAM scan via safety:: (Thorn Safer)
    │      Images: full scan
    │      Video: keyframe extraction + scan
    │      CSAM detected → QUARANTINE immediately, skip all remaining steps
    │                       Publish UploadQuarantined event
    │
    ├─ 4. Content moderation via safety:: (Rekognition)
    │      Images: full scan
    │      Video: keyframe extraction + scan
    │      Routing is determined by ModerationResult (set by SafetyScanBridge §11.2.2):
    │        auto_reject = true  → REJECT, store labels, skip remaining steps
    │                               Publish UploadRejected event
    │        has_violations = true, auto_reject = false
    │                            → FLAG, store labels, skip remaining steps
    │                               Publish UploadFlagged event
    │        has_violations = false → clean, continue pipeline
    │
    ├─ 5. Compression decision
    │      Compare probe results against compression profiles (§10.2)
    │      If thresholds exceeded → dispatch CompressAssetJob
    │      If within thresholds → skip to step 7
    │      (Many uploads skip this — modern cameras produce reasonable output)
    │
    ├─ 6. [Conditional] FFMPEG worker compression
    │      CompressAssetJob runs on FFMPEG worker (stateless service)
    │      Reads source from S3, compresses per profile, writes back to S3
    │      Replaces original at same storage key
    │      On completion → resume at step 7
    │
    ├─ 7. Variant generation (images only, unconditional)
    │      Generate thumb (200x200) and medium (800x800) via image processing library
    │      Runs in-process — no external worker needed (milliseconds)
    │      Operates on final file (post-compression if compressed, else original)
    │      Non-image assets: skip this step
    │
    └─ 8. Publish
           Set status → published
           Record final file size, set has_thumb/has_medium flags
           Set published_at timestamp
           Publish UploadPublished event
```

**Pipeline short-circuits**:
- CSAM detected at step 3 -> quarantine immediately, skip steps 4-8
- Moderation violation at step 4 -> flag, skip steps 5-8
- Not an image at step 7 -> skip variant generation (no variants for PDFs, audio, etc.)

For `video_lesson` context uploads, an additional `TranscodeVideoJob` step runs after the
standard processing pipeline completes. This job converts the raw video to HLS format
(see §10.8 Video Transcoding Pipeline).

**Key separation of concerns**:
- `CompressAssetJob` = **conditional**, heavy, external worker (FFMPEG). Only for oversized assets.
- Variant generation = **unconditional** (for images), lightweight, in-process. Always runs.

### §10.2 Compression Profiles

Compression is triggered **only when assets exceed quality thresholds**. Many uploads from
modern phones and cameras are already well-compressed and will pass through without recompression.

| Asset Type | Trigger Threshold | Target | Tool |
|------------|------------------|--------|------|
| **JPEG** | > 1.5 bytes/pixel | Quality 85, strip EXIF (preserve orientation) | Image library or FFMPEG |
| **PNG** | > 4 bytes/pixel | Re-encode as optimized PNG or convert to WebP | FFMPEG |
| **WebP** | > 1.0 bytes/pixel | Quality 82 | FFMPEG |
| **GIF** | > 5 MB | Re-encode, optimize palette | FFMPEG |
| **Video (AV1)** | > 4 Mbps for 1080p, > 2 Mbps for 720p | CRF 30, preset 6, AV1 via SVT-AV1 (`libsvtav1`) | FFMPEG |
| **Video (non-AV1 codec)** | Always | Transcode to AV1, CRF 30, preset 6 | FFMPEG |
| **Audio (uncompressed)** | WAV, FLAC, AIFF | AAC 192 kbps | FFMPEG |
| **Audio (compressed)** | > 256 kbps | AAC 192 kbps | FFMPEG |
| **PDF** | Never | No compression (pass-through) | — |

**Bytes-per-pixel** = `file_size_bytes / (width * height)`. Video bitrate from ffprobe
`bit_rate` field. Audio bitrate from ffprobe `bit_rate` field.

### §10.3 CSAM Scan Step

CSAM scanning is **mandatory** for all image and video uploads before publication. `[S§12.1]`

```go
// Within ProcessUploadJob pipeline
csamResult, err := s.safety.ScanCSAM(ctx, upload.StorageKey)

if err != nil {
    var scanErr *ScanError
    if errors.As(err, &scanErr) && scanErr.Code == "unavailable" {
        // Graceful degradation: log warning, continue processing
        // Upload will be flagged for manual review
        slog.Warn("CSAM scan unavailable — flagging for manual review",
            "upload_id", upload.ID)
        // Do NOT block publication — but flag for review
    } else {
        return fmt.Errorf("CSAM scan failed: %w", err)
    }
} else if csamResult.IsCSAM {
    // Quarantine immediately — do NOT process further
    if _, err := s.uploads.UpdateStatus(ctx, upload.ID, UploadStatusQuarantined,
        &UploadStatusUpdate{}); err != nil {
        return fmt.Errorf("quarantining upload: %w", err)
    }

    // Report to NCMEC via safety:: [18 U.S.C. § 2258A]
    if err := s.safety.ReportCSAM(ctx, upload.ID, csamResult); err != nil {
        return fmt.Errorf("reporting CSAM: %w", err)
    }

    // Publish event for safety:: to handle account suspension
    s.events.Publish(ctx, &UploadQuarantined{
        UploadID: upload.ID,
        FamilyID: upload.FamilyID,
        Context:  upload.Context,
    })

    return nil // Short-circuit — no further processing
}
```

### §10.4 Content Moderation Step

Content moderation uses AWS Rekognition to detect policy violations (explicit content,
violence, etc.). The `ModerationResult` returned by `SafetyScanBridge` includes routing
decisions (see `[11-safety §11.2.2]`): `AutoReject` for nudity/explicit content, flagging
for suggestive/violent content, and ignored categories (drugs, hate symbols, weapons) that
are legitimate educational content on a homeschool platform. `[S§12.2]`

```go
// Within ProcessUploadJob pipeline (after CSAM scan passes)
modResult, err := s.safety.ScanModeration(ctx, upload.StorageKey)

if err != nil {
    var scanErr *ScanError
    if errors.As(err, &scanErr) && scanErr.Code == "unavailable" {
        slog.Warn("Moderation scan unavailable — continuing",
            "upload_id", upload.ID)
    } else {
        return fmt.Errorf("moderation scan failed: %w", err)
    }
} else if modResult.AutoReject {
    // Auto-reject — content policy violation (e.g. nudity) [11-safety §11.2.1]
    labelsJSON, _ := json.Marshal(modResult.Labels)
    if err := s.uploads.SetModerationLabels(ctx, upload.ID, labelsJSON); err != nil {
        return fmt.Errorf("setting moderation labels: %w", err)
    }
    if _, err := s.uploads.UpdateStatus(ctx, upload.ID, UploadStatusRejected,
        &UploadStatusUpdate{}); err != nil {
        return fmt.Errorf("rejecting upload: %w", err)
    }

    s.events.Publish(ctx, &UploadRejected{
        UploadID: upload.ID,
        FamilyID: upload.FamilyID,
        Context:  upload.Context,
        Labels:   modResult.Labels,
    })

    return nil // Short-circuit
} else if modResult.HasViolations {
    // Flag for review — admin must assess
    labelsJSON, _ := json.Marshal(modResult.Labels)
    if err := s.uploads.SetModerationLabels(ctx, upload.ID, labelsJSON); err != nil {
        return fmt.Errorf("setting moderation labels: %w", err)
    }
    if _, err := s.uploads.UpdateStatus(ctx, upload.ID, UploadStatusFlagged,
        &UploadStatusUpdate{}); err != nil {
        return fmt.Errorf("flagging upload: %w", err)
    }

    s.events.Publish(ctx, &UploadFlagged{
        UploadID: upload.ID,
        FamilyID: upload.FamilyID,
        Context:  upload.Context,
        Labels:   modResult.Labels,
        Priority: modResult.Priority,
    })

    return nil // Short-circuit
}
```

### §10.5 ffprobe Metadata Schema

Stored as `probe_metadata JSONB` on `media_uploads`. Schema varies by asset type:

**Image**:
```json
{
    "format": "jpeg",
    "width": 3024,
    "height": 4032,
    "bytes_per_pixel": 1.2,
    "was_compressed": false,
    "file_size_original": 14680064,
    "file_size_compressed": null
}
```

**Video**:
```json
{
    "format": "mp4",
    "codec": "h264",
    "width": 1920,
    "height": 1080,
    "duration_seconds": 342.5,
    "bitrate_bps": 12500000,
    "was_compressed": true,
    "file_size_original": 534123456,
    "file_size_compressed": 198765432,
    "compression_ratio": 2.69
}
```

**Audio**:
```json
{
    "format": "mp3",
    "codec": "mp3",
    "bitrate_bps": 320000,
    "duration_seconds": 185.3,
    "sample_rate": 44100,
    "channels": 2,
    "was_compressed": false,
    "file_size_original": 7412736
}
```

### §10.6 FFMPEG Worker Architecture

The FFMPEG worker is a **stateless service** that processes compression jobs:

- **Receives**: S3 storage key, compression profile parameters
- **Reads**: Source asset from S3
- **Processes**: Runs FFMPEG with profile-specific parameters
- **Writes**: Compressed asset back to S3 at the same storage key (replaces original)
- **Returns**: Completion status, final file sizes

The worker is invoked via job queue (hibiken/asynq). `media::` enqueues `CompressAssetJob`,
the worker picks it up, processes, writes the result back, and marks the job complete.
`ProcessUploadJob` resumes at step 7 (variant generation) after compression completes.

**Deployment options** (spec is infrastructure-agnostic):
- **Lambda**: Good for images and short videos (< 15 min timeout, 10 GB `/tmp`).
  Cost-effective for bursty workloads.
- **Dedicated container**: Required for large videos (> 15 min encode time).
  ECS task or sidecar container.
- **Hybrid**: Lambda for images/audio, container for video (selected by asset type
  at dispatch time).

### §10.7 Variant Generation (Images Only)

Variant generation is a **separate, unconditional step** that runs for all image uploads
after compression (if any) completes. It uses an image processing library **in-process** — no
external worker needed. `[ARCH §8.2]`

```go
// Within ProcessUploadJob, after compression (if any) completes
if strings.HasPrefix(upload.ContentType, "image/") {
    variants := []struct {
        variant ImageVariant
        maxW    int
        maxH    int
        suffix  string
    }{
        {ImageVariantThumb, 200, 200, "thumb"},
        {ImageVariantMedium, 800, 800, "medium"},
    }

    // Download the final image (post-compression or original)
    imageBytes, err := s.storage.GetObjectBytes(ctx, upload.StorageKey, 0, uint64(upload.SizeBytes))
    if err != nil {
        return fmt.Errorf("downloading image for variants: %w", err)
    }

    for _, v := range variants {
        resized, err := resizeFitWithin(imageBytes, v.maxW, v.maxH, 85)
        if err != nil {
            return fmt.Errorf("generating %s variant: %w", v.suffix, err)
        }
        baseKey := strings.TrimSuffix(upload.StorageKey, filepath.Ext(upload.StorageKey))
        variantKey := fmt.Sprintf("%s__%s.%s", baseKey, v.suffix, variantExtension(upload.ContentType))
        if err := s.storage.PutObject(ctx, variantKey, resized, upload.ContentType); err != nil {
            return fmt.Errorf("uploading %s variant: %w", v.suffix, err)
        }
    }

    if err := s.uploads.SetVariantFlags(ctx, upload.ID, true, true); err != nil {
        return fmt.Errorf("setting variant flags: %w", err)
    }
}
```

- Variant keys: `{storage_key}__thumb.{ext}`, `{storage_key}__medium.{ext}`
- Non-image assets (video, audio, PDF): no variants generated in Phase 1
- Video variants (lower-resolution streams for adaptive playback): Phase 2+

### §10.8 Video Transcoding Pipeline (Phase 1) `[S§8.1.11]`

The video transcoding pipeline converts uploaded video files into HLS (HTTP Live Streaming)
format for adaptive bitrate delivery.

#### Upload Flow

1. Creator initiates video upload via `POST /v1/media/uploads` with context `video_lesson`
2. `media::` generates presigned upload URL, creator uploads raw video to R2
3. On upload confirmation, `ProcessUploadJob` runs standard pipeline (magic byte validation,
   ffprobe analysis, CSAM scan)
4. After passing safety checks, a `TranscodeVideoJob` is enqueued

#### Transcode Job

1. FFmpeg reads raw video from R2
2. Generates HLS segments at 2-3 quality levels:
   - **480p** (baseline): ~1 Mbps, suitable for mobile/slow connections
   - **720p** (standard): ~2.5 Mbps, good balance of quality and bandwidth
   - **1080p** (high): ~5 Mbps, full quality (only if source >= 1080p)
3. Generates master HLS playlist (`master.m3u8`) referencing quality variants
4. Uploads all segments and playlists to R2 under structured keys:
   `media/video/{upload_id}/master.m3u8`
   `media/video/{upload_id}/480p/segment_{n}.ts`
   `media/video/{upload_id}/720p/segment_{n}.ts`
   `media/video/{upload_id}/1080p/segment_{n}.ts`
5. Updates `media_transcode_jobs` with output keys and duration

#### Delivery

- Video player requests signed URL for `master.m3u8`
- Player automatically selects quality variant based on bandwidth
- CDN caches segments for subsequent viewers
- Signed URLs expire after configurable duration (default: 4 hours)

#### External Video Support

- YouTube/Vimeo videos store the external URL and metadata only — no transcoding
- `learn_video_defs.video_source` distinguishes `self_hosted` from `youtube`/`vimeo`
- External video progress tracked via JS API integration (YouTube IFrame API, Vimeo Player SDK)

#### Context Validation Rules for `video_lesson`

| Attribute | Value |
|-----------|-------|
| **Max file size** | 5 GB |
| **Allowed MIME types** | `video/mp4`, `video/quicktime`, `video/x-msvideo`, `video/webm` |
| **Magic byte validation** | Required — validate video container signatures |
| **Processing pipeline** | Standard + TranscodeVideoJob |

---

## §11 Magic Byte Validation (Deep-Dive 3)

Per `[CODING §5.2]`, all uploads MUST be validated by **file magic bytes** — not just
the `Content-Type` header or file extension. The declared `content_type` from the upload
request is used only for pre-flight validation and presigned URL generation. After the
file is actually uploaded to S3, `ProcessUploadJob` reads the first bytes and verifies
the file signature matches.

### §11.1 Supported File Signatures

| File Type | Magic Bytes | Offset |
|-----------|------------|--------|
| JPEG | `FF D8 FF` | 0 |
| PNG | `89 50 4E 47 0D 0A 1A 0A` | 0 |
| WebP | `52 49 46 46 ... 57 45 42 50` | 0 (RIFF at 0, WEBP at 8) |
| GIF | `47 49 46 38` | 0 |
| PDF | `25 50 44 46` | 0 |
| MP4 | `66 74 79 70` (ftyp) | 4 |
| MOV/QuickTime | `66 74 79 70` (ftyp, same container as MP4) | 4 |
| AVI | `52 49 46 46 ... 41 56 49 20` | 0 (RIFF at 0, AVI\x20 at 8) |
| WebM | `1A 45 DF A3` (EBML header) | 0 |
| MP3 | `FF FB` or `FF F3` or `FF F2` or `49 44 33` (ID3) | 0 |
| AAC/M4A | `66 74 79 70` (ftyp, same as MP4) | 4 |
| WAV | `52 49 46 46 ... 57 41 56 45` | 0 (RIFF at 0, WAVE at 8) |
| FLAC | `66 4C 61 43` | 0 |
| AIFF | `46 4F 52 4D ... 41 49 46 46` | 0 (FORM at 0, AIFF at 8) |
| ZIP | `50 4B 03 04` | 0 |

### §11.2 Validation Implementation

```go
// ValidateMagicBytes validates file magic bytes against declared content_type.
//
// Reads the first 16 bytes from S3 and matches against known file
// signatures. Returns nil if magic bytes match the declared type,
// or ErrInvalidFileType if they don't. [CODING §5.2]
func ValidateMagicBytes(ctx context.Context, storage ObjectStorageAdapter, storageKey string, declaredContentType string) error {
    header, err := storage.GetObjectBytes(ctx, storageKey, 0, 16)
    if err != nil {
        return fmt.Errorf("failed to read file header: %w", err)
    }

    detectedType := detectFileType(header)

    if !isCompatible(declaredContentType, detectedType) {
        return ErrInvalidFileType
    }

    return nil
}

// detectFileType detects file type from magic bytes. Returns a general MIME category.
func detectFileType(header []byte) string {
    if len(header) < 3 {
        return ""
    }

    // JPEG: FF D8 FF
    if header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
        return "image/jpeg"
    }

    // PNG: 89 50 4E 47 0D 0A 1A 0A
    if len(header) >= 8 && bytes.Equal(header[0:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
        return "image/png"
    }

    // GIF: 47 49 46 38
    if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x47, 0x49, 0x46, 0x38}) {
        return "image/gif"
    }

    // PDF: 25 50 44 46
    if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x25, 0x50, 0x44, 0x46}) {
        return "application/pdf"
    }

    // WebP: RIFF....WEBP
    if len(header) >= 12 &&
        bytes.Equal(header[0:4], []byte{0x52, 0x49, 0x46, 0x46}) &&
        bytes.Equal(header[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
        return "image/webp"
    }

    // WAV: RIFF....WAVE
    if len(header) >= 12 &&
        bytes.Equal(header[0:4], []byte{0x52, 0x49, 0x46, 0x46}) &&
        bytes.Equal(header[8:12], []byte{0x57, 0x41, 0x56, 0x45}) {
        return "audio/wav"
    }

    // AIFF: FORM....AIFF
    if len(header) >= 12 &&
        bytes.Equal(header[0:4], []byte{0x46, 0x4F, 0x52, 0x4D}) &&
        bytes.Equal(header[8:12], []byte{0x41, 0x49, 0x46, 0x46}) {
        return "audio/aiff"
    }

    // FLAC: fLaC
    if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x66, 0x4C, 0x61, 0x43}) {
        return "audio/flac"
    }

    // MP3: FF FB/F3/F2 or ID3 tag
    if (header[0] == 0xFF && (header[1]&0xE0) == 0xE0) ||
        (len(header) >= 3 && bytes.Equal(header[0:3], []byte{0x49, 0x44, 0x33})) {
        return "audio/mpeg"
    }

    // WebM: EBML header 1A 45 DF A3
    if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x1A, 0x45, 0xDF, 0xA3}) {
        return "video/webm"
    }

    // MP4/M4A/MOV: ftyp box at offset 4
    if len(header) >= 8 && bytes.Equal(header[4:8], []byte{0x66, 0x74, 0x79, 0x70}) {
        return "video/mp4"
    }

    // AVI: RIFF....AVI\x20
    if len(header) >= 12 &&
        bytes.Equal(header[0:4], []byte{0x52, 0x49, 0x46, 0x46}) &&
        bytes.Equal(header[8:12], []byte{0x41, 0x56, 0x49, 0x20}) {
        return "video/x-msvideo"
    }

    // ZIP: PK\x03\x04
    if len(header) >= 4 && bytes.Equal(header[0:4], []byte{0x50, 0x4B, 0x03, 0x04}) {
        return "application/zip"
    }

    return ""
}
```

### §11.3 Mismatch Handling

If magic bytes don't match the declared content type:
1. Update upload status to `rejected` with reason `invalid_magic_bytes`
2. Log the mismatch internally (declared type vs. detected type) — `[CODING §2.2, §5.2]` never expose in API
3. Return `ErrMagicByteMismatch` to the processing pipeline (internal error, not exposed via HTTP)
4. The upload is NOT published — it remains in `rejected` status and is not recoverable

---

## §12 Orphan Cleanup (Deep-Dive 4)

Orphan uploads are presigned URLs that were requested but never completed (the client
crashed, navigated away, or abandoned the upload). These leave `pending` records in
`media_uploads` with no corresponding S3 object (or an incomplete upload).

### §12.1 CleanupOrphanUploadsJob

- **Queue**: Low `[ARCH §12.2]`
- **Schedule**: Daily at 4:00 AM UTC
- **Logic**:
  1. Find all uploads with `status = 'pending'` AND `expires_at < now()`
  2. For each orphan:
     a. Attempt to delete the S3 object (may not exist — that's OK)
     b. Transition status to `expired`
  3. Log count of cleaned-up orphans

```go
func RunCleanup(ctx context.Context, uploads UploadRepository, storage ObjectStorageAdapter) (uint32, error) {
    orphans, err := uploads.FindExpiredPending(ctx, time.Now(), 1000)
    if err != nil {
        return 0, fmt.Errorf("finding expired uploads: %w", err)
    }

    var cleaned uint32
    for _, orphan := range orphans {
        // Best-effort S3 deletion — object may not exist
        _ = storage.DeleteObject(ctx, orphan.StorageKey)

        if _, err := uploads.UpdateStatus(ctx, orphan.ID, UploadStatusExpired,
            &UploadStatusUpdate{}); err != nil {
            // Continue on error — don't let one failed update block the entire batch.
            // The orphan will be retried on the next scheduled run.
            slog.Error("failed to expire orphan upload", "upload_id", orphan.ID, "error", err)
            continue
        }

        cleaned++
    }

    if cleaned > 0 {
        slog.Info("cleaned up orphan uploads", "count", cleaned)
    }

    return cleaned, nil
}
```

### §12.2 Recurring Schedule

```go
// Added to the shared scheduler [ARCH §12.3]
scheduler.Register("0 4 * * *", NewCleanupOrphanUploadsTask()) // Daily at 4:00 AM UTC
```

---

## §13 CDN & Delivery Strategy (Deep-Dive 5)

### §13.1 URL Generation

URLs for published media are **public CDN URLs** — no authentication or signing required.
The content is keyed by `upload_id` (UUID), making it effectively content-addressed and
safe to cache indefinitely. `[ARCH §2.10]`

```
Published media:     {OBJECT_STORAGE_PUBLIC_URL}/{storage_key}
Published thumb:     {OBJECT_STORAGE_PUBLIC_URL}/{storage_key}__thumb.{ext}
Published medium:    {OBJECT_STORAGE_PUBLIC_URL}/{storage_key}__medium.{ext}
Marketplace files:   presigned GET URL (never public — purchase verification required)
```

### §13.2 Cache Headers

Published media is immutable (identified by UUID) — cache forever:
```
Cache-Control: public, max-age=31536000, immutable
```

### §13.3 Provider Agnosticism

- The public URL base is configured via `OBJECT_STORAGE_PUBLIC_URL` env var — fully
  decoupled from the storage provider
- CDN is configured externally:
  - Cloudflare R2: automatic CDN via Cloudflare (zero configuration)
  - AWS S3: requires separate CloudFront distribution
  - MinIO: requires separate reverse proxy / CDN
- This spec explicitly avoids any R2-specific API calls — only standard S3 operations
- Switching providers requires only environment variable changes:
  - `OBJECT_STORAGE_ENDPOINT` -> new provider endpoint
  - `OBJECT_STORAGE_PUBLIC_URL` -> new CDN base URL
  - Credentials -> new provider credentials

---

## §14 Background Jobs

| Job | Queue | Trigger | Description |
|-----|-------|---------|-------------|
| `ProcessUploadJob` | Default | Upload confirmation (`ConfirmUpload`) | Orchestrator: magic bytes -> ffprobe -> CSAM -> moderation -> compression decision -> variant generation -> publish `[ARCH §12.2]` |
| `CompressAssetJob` | Default | Dispatched by `ProcessUploadJob` **only if thresholds exceeded** | Sent to FFMPEG worker: compress asset per profile, write back to S3. Does NOT generate variants. |
| `TranscodeVideoJob` | Default | Dispatched by `ProcessUploadJob` for `video_lesson` context uploads after safety checks pass | Converts raw video to HLS adaptive bitrate format (480p/720p/1080p segments + master playlist). See §10.8. |
| `CleanupOrphanUploadsJob` | Low | Daily at 4:00 AM UTC | Expire pending uploads past presigned URL expiry `[ARCH §12.3]` |

> **Note**: Variant generation (thumb + medium) is NOT a separate job — it runs in-process
> within `ProcessUploadJob` after compression completes (or immediately if no compression
> needed). It uses an image processing library and takes milliseconds.

---

## §15 Error Types

```go
// internal/media/errors.go

// Media domain error types. [CODING §2.2, §5.2]
//
// Internal error details (storage errors, scan failures) are logged
// but NEVER exposed in API responses. The HTTP mapping returns only
// generic user-facing messages.

import "errors"

var (
    // ─── Upload lifecycle ───────────────────────────────────────────────
    ErrUploadNotFound    = errors.New("upload not found")
    ErrInvalidFileType   = errors.New("invalid file type for this context")
    ErrFileTooLarge      = errors.New("file exceeds maximum size for this context")
    ErrUploadNotConfirmed = errors.New("upload has not been confirmed")
    ErrUploadQuarantined = errors.New("upload is quarantined")
    ErrUploadRejected    = errors.New("upload was rejected by content policy")
    ErrUploadFlagged     = errors.New("upload is flagged for review")
    ErrUploadExpired     = errors.New("upload has expired")
    ErrNotOwner          = errors.New("not the upload owner")

    // ─── External service errors ────────────────────────────────────────
    ErrObjectStorageError = errors.New("object storage operation failed")
    ErrScanServiceUnavailable = errors.New("safety scan service unavailable")
    ErrScanServiceFailed = errors.New("safety scan failed")
)
```

### §15.1 HTTP Status Code Mapping

| Error | HTTP Status | Error Code | User-Facing Message |
|-------|-------------|------------|---------------------|
| `ErrUploadNotFound` | 404 | `upload_not_found` | "Upload not found" |
| `ErrInvalidFileType` | 422 | `invalid_file_type` | "File type is not allowed for this upload context" |
| `ErrFileTooLarge` | 422 | `file_too_large` | "File exceeds the maximum allowed size" |
| `ErrUploadNotConfirmed` | 409 | `upload_not_confirmed` | "Upload must be confirmed before this operation" |
| `ErrUploadQuarantined` | 403 | `upload_quarantined` | "This upload has been restricted" |
| `ErrUploadRejected` | 403 | `upload_rejected` | "This upload was not published because it violates our content guidelines" |
| `ErrUploadFlagged` | 403 | `upload_flagged` | "This upload is under review" |
| `ErrUploadExpired` | 410 | `upload_expired` | "Upload link has expired — please request a new one" |
| `ErrNotOwner` | 403 | `not_owner` | "You do not have permission to access this upload" |
| `ErrObjectStorageError` | 502 | `storage_error` | "File storage is temporarily unavailable" |
| `ErrScanServiceUnavailable` | 503 | `scan_unavailable` | "Content scanning is temporarily unavailable" |
| `ErrScanServiceFailed` | 502 | `scan_failed` | "Content scanning encountered an error" |

> **Note**: Internal error details are logged via `slog.Error` but the API response contains
> only the generic user-facing message. `[CODING §2.2, §5.2]`

---

## §16 Cross-Domain Interactions

### §16.1 media:: Provides (Consumed by Other Domains)

| Consumer | Interface | Usage |
|----------|-----------|-------|
| `social::` | `MediaService.RequestUpload()` | Post/message attachment uploads `[05-social §4.1]` |
| `social::` | `MediaService.ValidateAttachment()` | Pre-flight attachment validation |
| `learn::` | `MediaService.RequestUpload()` | Activity, journal, project attachment uploads `[06-learn §7]` |
| `learn::` | `MediaService.ValidateAttachment()` | Attachment validation (replaces `learn.MediaAdapter.ValidateAttachment`) |
| `mkt::` | `MediaService.RequestUpload()` | Listing file, preview, thumbnail uploads `[07-mkt §7]` |
| `mkt::` | `MediaService.PresignedGet()` | Purchased file download URLs (replaces `mkt.MediaAdapter.PresignedGet`) `[ARCH §8.3]` |
| `search::` | `UploadPublished` event | Index media metadata for search |
| `safety::` | `UploadQuarantined` event | NCMEC report trigger, account suspension |
| `safety::` | `UploadRejected` event | Auto-rejected content flag + user notification `[11-safety §11.2.1]` |
| `safety::` | `UploadFlagged` event | Moderation queue entry |

### §16.2 media:: Consumes

| Provider | Interface | Usage |
|----------|-----------|-------|
| `iam::` | `AuthContext` middleware | Authentication and authorization `[00-core §7.2]` |
| `iam::` | `FamilyScope` middleware | Family-scoped data access `[00-core §8]` |
| `safety::` | `SafetyScanAdapter.ScanCSAM()` | CSAM hash matching via Thorn Safer `[ARCH §2.13]` |
| `safety::` | `SafetyScanAdapter.ScanModeration()` | Content moderation via Rekognition `[ARCH §2.13]` |
| `safety::` | `SafetyScanAdapter.ReportCSAM()` | NCMEC reporting `[S§12.1]` |

### §16.3 Domain Events Published

```go
// UploadPublished is published when an upload completes all processing and is ready for use.
// Consumed by search:: (index metadata) and the originating domain
// (update attachment status if needed).
type UploadPublished struct {
    UploadID    uuid.UUID
    FamilyID    uuid.UUID
    Context     UploadContext
    StorageKey  string
    ContentType string
    SizeBytes   int64  // actual file size from S3 HEAD
    HasThumb    bool   // true if a thumbnail variant was generated
    HasMedium   bool   // true if a medium variant was generated
}

// UploadQuarantined is published when CSAM is detected in an upload.
// Consumed by safety:: for NCMEC reporting and account suspension.
// [S§12.1, 18 U.S.C. § 2258A]
type UploadQuarantined struct {
    UploadID uuid.UUID
    FamilyID uuid.UUID
    Context  UploadContext
}

// UploadRejected is published when content moderation auto-rejects an upload (e.g. nudity).
// Consumed by safety:: (creates content flag + user rejection notification).
// [S§12.2, 11-safety §11.2.1]
type UploadRejected struct {
    UploadID uuid.UUID
    FamilyID uuid.UUID
    Context  UploadContext
    Labels   []ModerationLabel
}

// UploadFlagged is published when content moderation flags an upload for admin review.
// Consumed by safety:: for moderation queue.
// [S§12.2, 11-safety §11.2.2]
type UploadFlagged struct {
    UploadID uuid.UUID
    FamilyID uuid.UUID
    Context  UploadContext
    Labels   []ModerationLabel
    Priority *string // "critical", "high", "normal" — from label routing
}
```

### §16.4 Domain Events Subscribed To

None in Phase 1. `media::` is a producer (publishes upload lifecycle events), not a consumer.
Future phases may subscribe to `FamilyDeletionScheduled` from `iam::` to cascade-delete all
family media.

### §16.5 Adapter Interface Reconciliation

The following adapter interfaces in existing domain specs are **superseded** by `media.MediaService`:

| Existing Sketch | Location | Replacement |
|----------------|----------|-------------|
| `mkt.MediaAdapter.PresignedUpload()` | `07-mkt §7` | `media.MediaService.RequestUpload()` |
| `mkt.MediaAdapter.PresignedGet()` | `07-mkt §7` | `media.MediaService.PresignedGet()` |
| `learn.MediaAdapter.ValidateAttachment()` | `06-learn §7` | `media.MediaService.ValidateAttachment()` |
| `learn.MediaAdapter.GetUploadURL()` | `06-learn §7` | `media.MediaService.RequestUpload()` |

Both `07-mkt` and `06-learn` already note that their `MediaAdapter` interfaces are consumed
from `media::` — this spec makes the contract authoritative. Implementation injects
a `media.MediaService` interface value where those specs reference `MediaAdapter`.

---

## §17 Phase Scope

### Phase 1 (MVP)

- **API**: 3 endpoints (request upload, confirm upload, get upload)
- **Processing**: `ProcessUploadJob` — magic byte validation, ffprobe analysis, CSAM scan,
  content moderation, compression decision + `CompressAssetJob`, image variant generation
  (thumb + medium), publish
- **Storage**: S3-compatible adapter (`internal/media/adapters/s3.go`)
- **Validation**: Context-based size/type rules, magic byte verification
- **Safety**: CSAM scan (Thorn Safer), content moderation (Rekognition) — both via `safety::` adapters
- **Compression**: FFMPEG worker for all asset types (image, video, audio)
- **Variants**: Thumb (200x200) and medium (800x800) for images only
- **Events**: `UploadPublished`, `UploadQuarantined`, `UploadRejected`, `UploadFlagged`

### Phase 2

- **API**: Delete upload, list uploads, admin reprocess
- **Jobs**: `CleanupOrphanUploadsJob` (orphan expiry)
- **Features**: Compression analytics dashboard, family deletion cascade
- **Variants**: Video adaptive variants (lower-resolution streams)

### Phase 3

- **API**: Bulk upload
- **Features**: Storage quota enforcement (per tier, via `billing::`), CDN cache invalidation
  API, video adaptive streaming (HLS/DASH)

---

## §18 Verification Checklist

### Upload Flow

1. `POST /v1/media/uploads` validates context, content_type, and size_bytes before generating
   a presigned URL
2. Presigned URL includes Content-Type and Content-Length constraints
3. `POST /v1/media/uploads/:id/confirm` verifies S3 object exists via HEAD before transitioning
   status
4. Confirm endpoint updates `size_bytes` from actual S3 object size (not declared value)
5. Upload status transitions follow the state machine in §3.1 — no invalid transitions

### Image Processing

6. `ProcessUploadJob` generates both thumb (200x200) and medium (800x800) variants for all
   image uploads `[ARCH §8.2]`
7. Variants are generated using an image processing library in-process (not external worker)
8. Variant keys follow the pattern `{storage_key}__thumb.{ext}` and `{storage_key}__medium.{ext}`
9. Non-image uploads skip variant generation
10. Compression profiles are applied only when thresholds are exceeded (§10.2)

### CSAM & Moderation

11. All image/video uploads are scanned for CSAM before publication `[S§12.1]`
12. CSAM detection triggers immediate quarantine — no further processing `[S§12.1]`
13. CSAM is reported to NCMEC via `safety.Service.ReportCSAM()` `[18 U.S.C. § 2258A]`
14. Content moderation runs on all image/video uploads `[S§12.2]`
15. Nudity/explicit labels above threshold -> upload auto-rejected (not just flagged) `[11-safety §11.2.1]`
16. Suggestive/violence labels -> upload flagged for admin review `[11-safety §11.2.2]`
17. Drugs/tobacco/alcohol, hate symbols, weapons -> ignored (legitimate educational content)
18. Rejected uploads are appealable; quarantined (CSAM) uploads are not
19. `UploadQuarantined`, `UploadRejected`, and `UploadFlagged` events are published for `safety::` consumption

### Signed URLs

20. Upload presigned URLs expire after 1 hour
21. Download presigned URLs expire after 1 hour (marketplace files only)
22. Published media uses public CDN URLs (no signing required)
23. Marketplace files always use presigned GET URLs (never public) `[ARCH §8.3]`

### Orphan Cleanup

24. `CleanupOrphanUploadsJob` runs daily at 4:00 AM UTC `[ARCH §12.3]`
25. Only `pending` uploads past `expires_at` are cleaned up
26. S3 object deletion is best-effort (object may not exist)
27. Cleaned-up uploads transition to `expired` status

### Error Handling

28. All errors use custom error types with `errors.Is`/`errors.As` `[CODING §2.2, §5.2]`
29. Internal error details (storage errors, scan failures) are logged but never exposed
    in API responses `[CODING §2.2, §5.2]`
30. `ErrScanServiceUnavailable` returns 503 — graceful degradation if Thorn/Rekognition is down

### Privacy Invariants

31. Every `media_uploads` query is family-scoped via `FamilyScope` `[CODING §2.4]`
32. Upload owner is verified before confirm/delete operations
33. Storage keys are namespaced by `family_id` — no cross-family access
34. EXIF data stripped during JPEG compression (preserving orientation only)
35. No GPS coordinates stored or exposed `[ARCH §1.5]`

### Cross-Domain Contracts

36. `MediaService` interface is the authoritative interface — supersedes `learn.MediaAdapter` and
    `mkt.MediaAdapter` sketches
37. `social::`, `learn::`, and `mkt::` inject `media.MediaService` interface values
38. Marketplace file downloads use `PresignedGet()` with 1-hour expiry `[ARCH §8.3]`
39. File versioning is `mkt::`'s responsibility — `media::` manages individual uploads only

---

## §19 Module Structure

```
internal/media/
├── handler.go            # HTTP handlers (3 Phase 1 endpoints)
│                         #   RequestUpload, ConfirmUpload, GetUpload
├── service.go            # MediaServiceImpl — orchestration between repo, storage,
│                         #   safety adapters, and event bus
├── repository.go         # GormUploadRepository, GormProcessingJobRepository
│                         #   All user-data queries family-scoped via FamilyScope
├── models.go             # GORM models, request/response types,
│                         #   internal types (UploadContext, UploadStatus, etc.)
├── validation.go         # Magic byte validation (§11), context-based rules (§9.1),
│                         #   filename sanitization
├── compression.go        # Compression profiles (§10.2), ffprobe analysis,
│                         #   threshold logic, variant generation
├── ports.go              # MediaService, UploadRepository, ProcessingJobRepository,
│                         #   ObjectStorageAdapter, SafetyScanAdapter interfaces
├── errors.go             # MediaError sentinel errors
├── events.go             # UploadPublished, UploadQuarantined, UploadRejected, UploadFlagged
├── jobs.go               # ProcessUploadJob, CompressAssetJob, CleanupOrphanUploadsJob
│                         #   Job definitions and handlers [ARCH §12.2]
└── adapters/
    └── s3.go             # S3-compatible object storage client (provider-agnostic)
                          #   Uses aws-sdk-go-v2 with custom endpoint URL [ARCH §2.10]
```

> **Complexity class**: Simple (no `domain/` subdirectory). `media::` has straightforward
> pipeline orchestration without the complex domain invariants that warrant aggregate models.
> `[ARCH §4.5]`
