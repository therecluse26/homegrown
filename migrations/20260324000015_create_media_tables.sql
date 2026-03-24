-- +goose Up
-- +goose StatementBegin

-- ─── media_uploads ──────────────────────────────────────────────────────
-- Core upload tracking table. Every file that enters the system gets a row
-- here regardless of which domain initiated the upload. Family-scoped.
-- [S§2.1, ARCH §8.1, 09-media §3.2]
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
    content_type          TEXT NOT NULL,

    -- File metadata (populated after upload confirmation)
    size_bytes            BIGINT,

    -- Storage
    storage_key           TEXT NOT NULL UNIQUE,

    -- Variants (set after processing)
    has_thumb             BOOLEAN NOT NULL DEFAULT false,
    has_medium            BOOLEAN NOT NULL DEFAULT false,

    -- Probe / compression metadata (populated by ProcessUploadJob)
    probe_metadata        JSONB,
    original_size_bytes   BIGINT,
    was_compressed        BOOLEAN NOT NULL DEFAULT false,

    -- Moderation (populated by safety:: scan results)
    moderation_labels     JSONB,
    last_csam_scanned_at  TIMESTAMPTZ,

    -- Lifecycle
    expires_at            TIMESTAMPTZ,
    published_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary query pattern: list uploads for a family filtered by context
CREATE INDEX idx_media_uploads_family_context
    ON media_uploads(family_id, context);

-- FK index on uploaded_by [CODING §4.3]
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
    error_message         TEXT,
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
CREATE TABLE media_transcode_jobs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id             UUID NOT NULL REFERENCES media_uploads(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    input_key             TEXT NOT NULL,
    output_keys           JSONB,
    resolutions           JSONB NOT NULL DEFAULT '["480p", "720p", "1080p"]',
    duration_seconds      INTEGER,
    error_message         TEXT,
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_media_transcode_jobs_upload ON media_transcode_jobs(upload_id);
CREATE INDEX idx_media_transcode_jobs_status ON media_transcode_jobs(status)
    WHERE status IN ('pending', 'processing');

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS media_transcode_jobs;
DROP TABLE IF EXISTS media_processing_jobs;
DROP TABLE IF EXISTS media_uploads;
