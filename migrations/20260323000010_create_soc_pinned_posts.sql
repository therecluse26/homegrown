-- +goose Up
-- soc_pinned_posts: tracks which posts are pinned in which groups. [05-social §4.2]
-- Moderator/owner only. Each group can pin multiple posts; each post can be pinned
-- in at most one group (enforced by unique constraint on group_id + post_id).
CREATE TABLE IF NOT EXISTS soc_pinned_posts (
    id          UUID        NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    group_id    UUID        NOT NULL REFERENCES soc_groups(id)  ON DELETE CASCADE,
    post_id     UUID        NOT NULL REFERENCES soc_posts(id)   ON DELETE CASCADE,
    pinned_by   UUID        NOT NULL,
    pinned_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_soc_pinned_posts_group_post UNIQUE (group_id, post_id)
);

CREATE INDEX idx_soc_pinned_posts_group ON soc_pinned_posts(group_id);

-- +goose Down
DROP TABLE IF EXISTS soc_pinned_posts;
