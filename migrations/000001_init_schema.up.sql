-- 000001_init_schema.up.sql
-- Initial Beast CTF platform database schema.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Teams
CREATE TABLE teams (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique team identifier
    name                VARCHAR(128) UNIQUE NOT NULL,                -- Team display name
    creator_id          UUID NULL,                                   -- User who originally created the team (FK added below)
    leader_id           UUID NULL,                                   -- Current team leader / captain (FK added below)
    invite_token        VARCHAR(64) UNIQUE NULL,                     -- Secret token used by members to join the team
    affiliation         VARCHAR(128) NULL,                           -- University, company, or organization affiliation
    country_code        CHAR(2) NULL,                                -- ISO 3166-1 alpha-2 country code (e.g. 'US', 'IN')
    is_banned           BOOLEAN DEFAULT FALSE,                       -- Disqualification toggle to hide from scoreboard
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),       -- Team registration timestamp
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- Team profile last update timestamp
);

-- Users
CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- User unique identifier (user_id)
    team_id             UUID NULL REFERENCES teams(id) ON DELETE SET NULL, -- Associated team UUID
    username            VARCHAR(64) UNIQUE NOT NULL,                 -- Unique handle / username
    email               VARCHAR(255) UNIQUE NOT NULL,                -- User email address
    password_hash       VARCHAR(255) NOT NULL,                       -- Argon2id or bcrypt password hash
    salt_uuid           UUID NOT NULL DEFAULT gen_random_uuid(),     -- Per-user cryptographic salt UUID
    role                VARCHAR(32) NOT NULL DEFAULT 'contestant' CHECK (role IN ('contestant', 'admin')), -- Role: contestant or admin
    public_ssh_key      TEXT NULL,                                   -- Public SSH key for container shell access
    is_verified         BOOLEAN DEFAULT FALSE,                       -- Email verification status flag
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),       -- Account creation timestamp
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- Account last update timestamp
);

ALTER TABLE teams ADD CONSTRAINT fk_teams_creator FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE teams ADD CONSTRAINT fk_teams_leader FOREIGN KEY (leader_id) REFERENCES users(id) ON DELETE RESTRICT;

-- Authentication OTPs
CREATE TABLE auth_otps (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique OTP transaction identifier
    user_id             UUID NULL REFERENCES users(id) ON DELETE CASCADE, -- User associated (NULL if verifying uncreated email)
    email               VARCHAR(255) NOT NULL,                       -- Email destination where OTP was sent
    otp_hash            CHAR(64) NOT NULL,                           -- SHA-256 hash of the 6-digit OTP code
    type                VARCHAR(32) NOT NULL CHECK (type IN ('email_verification', 'password_reset', 'login_2fa')),
    attempts            INT NOT NULL DEFAULT 0 CHECK (attempts >= 0), -- Failed attempt count; application locks further guesses at max_attempts
    max_attempts        INT NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    expires_at          TIMESTAMP WITH TIME ZONE NOT NULL,           -- Expiration timestamp (e.g. 10 minutes from creation)
    consumed_at         TIMESTAMP WITH TIME ZONE NULL,               -- Timestamp when successfully redeemed
    ip_address          INET NOT NULL,                               -- Client IP address that requested the OTP
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- Timestamp when OTP was generated
);

CREATE INDEX idx_auth_otps_lookup ON auth_otps (email, type, expires_at);

-- Challenges
CREATE TABLE challenges (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique challenge identifier
    name                VARCHAR(255) NOT NULL,                       -- Display title of the challenge
    slug                VARCHAR(255) UNIQUE NOT NULL,                -- URL-friendly unique identifier (e.g. 'baby-overflow')
    description         TEXT,                                        -- Challenge background and prompt in Markdown
    author              VARCHAR(128),                                -- Creator handle or author attribution
    max_points          INT NOT NULL CHECK (max_points >= min_points), -- Initial maximum point value for dynamic scoring
    min_points          INT NOT NULL CHECK (min_points >= 0),         -- Minimum floor points after decay
    decay_solves        INT NOT NULL DEFAULT 50 CHECK (decay_solves > 0), -- Number of solves required to reach min_points
    max_attempts        INT NULL CHECK (max_attempts > 0),            -- Submission attempt cap per team (NULL = unlimited)
    flag_template       VARCHAR(255) NULL,                           -- Format template for dynamic flags (e.g. 'CTF{seed_%s}')
    is_hidden           BOOLEAN DEFAULT FALSE,                       -- Visibility toggle (TRUE hides from scoreboard)
    release_at          TIMESTAMP WITH TIME ZONE NULL,               -- Scheduled unlock timestamp (NULL = available immediately)
    deployment_mode     VARCHAR(16) NOT NULL DEFAULT 'offline'
        CHECK (deployment_mode IN ('offline', 'shared', 'instanced')),
    generator_path      VARCHAR(512) NULL,                           -- Host executable path to generate individualized assets
    generator_timeout_s INT DEFAULT 30,                              -- Max execution time in seconds for asset generator
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),       -- Timestamp when challenge was created
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- Timestamp when challenge was last modified
);

-- Tags
CREATE TABLE tags (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(64) NOT NULL UNIQUE
);

CREATE TABLE challenge_tags (
    challenge_id        UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    tag_id              UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (challenge_id, tag_id)
);
CREATE INDEX idx_challenge_tags_tag ON challenge_tags (tag_id);

-- Prerequisites
CREATE TABLE challenge_prerequisites (
    challenge_id        UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE, -- Dependent challenge that remains locked
    requires_id         UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE, -- Prerequisite challenge that must be solved first
    PRIMARY KEY (challenge_id, requires_id),
    CHECK (challenge_id <> requires_id)
);

-- Hints
CREATE TABLE challenge_hints (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique hint identifier
    challenge_id        UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE, -- Target challenge
    content             TEXT NOT NULL,                               -- Markdown text of the hint
    cost_points         INT NOT NULL DEFAULT 0 CHECK (cost_points >= 0), -- Score penalty deducted when a team unlocks this hint
    release_at          TIMESTAMP WITH TIME ZONE NULL                -- Scheduled unlock time (NULL = available immediately)
);

CREATE TABLE user_hints (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique hint unlock transaction identifier
    hint_id             UUID NOT NULL REFERENCES challenge_hints(id) ON DELETE RESTRICT, -- Prohibit deleting hints that have been purchased
    team_id             UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,      -- Team paying for and unlocking the hint
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,      -- Individual user who unlocked the hint
    cost_points         INT NOT NULL CHECK (cost_points >= 0),        -- Points deducted from team score at transaction time
    unlocked_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),       -- Timestamp when hint was purchased
    UNIQUE (hint_id, team_id)
);

CREATE INDEX idx_user_hints_team ON user_hints (team_id, unlocked_at);

-- Challenge Files
CREATE TABLE challenge_files (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique downloadable file identifier
    challenge_id        UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE, -- Associated challenge
    team_id             UUID NULL REFERENCES teams(id) ON DELETE CASCADE, -- NULL = public/shared file; team UUID if generated per team
    display_name        VARCHAR(255) NOT NULL,                       -- Filename presented to participants (e.g. 'chall.zip')
    storage_path        VARCHAR(512) NOT NULL,                       -- File path on local storage or object bucket
    sha256              CHAR(64) NOT NULL,                           -- SHA-256 checksum for integrity check and cache busting
    size_bytes          BIGINT NOT NULL CHECK (size_bytes >= 0),      -- File size in bytes for download headers
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- File creation or upload timestamp
);

CREATE INDEX idx_challenge_files_team ON challenge_files (challenge_id, team_id);

-- Challenge Flags
CREATE TABLE challenge_flags (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique flag record identifier
    challenge_id        UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE, -- Associated challenge
    team_id             UUID NULL REFERENCES teams(id) ON DELETE CASCADE, -- NULL = static/shared flag; team UUID if generated dynamically
    flag_hash           CHAR(64) NOT NULL,                           -- Salted HMAC-SHA256 or bcrypt hash of the correct flag
    case_sensitive      BOOLEAN DEFAULT TRUE,                        -- Case sensitivity toggle (FALSE folds case before hashing)
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- Flag creation timestamp
);

CREATE INDEX idx_flags_hash ON challenge_flags (flag_hash);
CREATE INDEX idx_flags_lookup ON challenge_flags (challenge_id, team_id);

-- Challenge Runtime Config (One row per non-static challenge)
CREATE TABLE challenge_runtime_config (
    challenge_id        UUID PRIMARY KEY REFERENCES challenges(id) ON DELETE CASCADE,
    runtime_type        VARCHAR(16) NOT NULL CHECK (runtime_type IN ('image', 'compose')),
    image_ref           VARCHAR(512) NULL,                           -- Registry image reference; preferably pinned by digest, not a local image ID
    compose_path        VARCHAR(512) NULL,                           -- Trusted deployment configuration accessible to the worker
    CHECK (
        (runtime_type = 'image' AND image_ref IS NOT NULL AND compose_path IS NULL)
            OR
        (runtime_type = 'compose' AND compose_path IS NOT NULL AND image_ref IS NULL)
    )
);

-- Challenge Instance Config (One row per instanced challenge)
CREATE TABLE challenge_instance_config (
    challenge_id        UUID PRIMARY KEY REFERENCES challenges(id) ON DELETE CASCADE,
    ttl_minutes         INT NOT NULL DEFAULT 30 CHECK (ttl_minutes > 0),
    max_renewals        INT NOT NULL DEFAULT 2 CHECK (max_renewals >= 0)
);

-- Challenge Deployments
CREATE TABLE challenge_deployments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id        UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    owner_user_id       UUID NULL REFERENCES users(id) ON DELETE RESTRICT,         -- Participant who spawned the instance; NULL for shared deployments
    node_id             VARCHAR(64) NOT NULL DEFAULT 'local',
    status              VARCHAR(32) NOT NULL
        CHECK (status IN ('starting', 'running', 'stopping', 'stopped', 'expired', 'error')),
    runtime_identifier  VARCHAR(255) NULL,                           -- Container ID for image runtime; Compose project name for a stack
    instance_dir        VARCHAR(512) NULL,                           -- Worker filesystem path needed for cleanup
    renewals_used       INT NOT NULL DEFAULT 0 CHECK (renewals_used >= 0),
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at          TIMESTAMP WITH TIME ZONE NULL,
    stopped_at          TIMESTAMP WITH TIME ZONE NULL,

    cleaned_up_at       TIMESTAMP WITH TIME ZONE NULL,
    expires_at          TIMESTAMP WITH TIME ZONE NULL,               -- Required for user-owned instances; NULL for shared deployments
    last_renewed_at     TIMESTAMP WITH TIME ZONE NULL,

    CHECK (cleaned_up_at IS NULL OR status IN ('stopped', 'expired', 'error')),
    CHECK (
        (owner_user_id IS NULL AND expires_at IS NULL
            AND renewals_used = 0 AND last_renewed_at IS NULL)
            OR
        (owner_user_id IS NOT NULL AND expires_at IS NOT NULL)
    )
);

-- Helper function to resolve the owning team (or user if unteamed) for deployment slot uniqueness
CREATE OR REPLACE FUNCTION get_user_team_id(user_uuid UUID)
RETURNS UUID AS $$
    SELECT COALESCE(team_id, id) FROM users WHERE id = user_uuid;
$$ LANGUAGE sql IMMUTABLE;

CREATE UNIQUE INDEX uq_active_team_challenge_deployment
    ON challenge_deployments (challenge_id, get_user_team_id(owner_user_id))
    WHERE owner_user_id IS NOT NULL AND cleaned_up_at IS NULL;

CREATE UNIQUE INDEX uq_active_shared_challenge_deployment
    ON challenge_deployments (challenge_id)
    WHERE owner_user_id IS NULL AND cleaned_up_at IS NULL;

-- Deployment Endpoints
CREATE TABLE deployment_endpoints (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id       UUID NOT NULL REFERENCES challenge_deployments(id) ON DELETE CASCADE,
    name                VARCHAR(64) NOT NULL,                        -- Stable label, e.g. web or pwn
    protocol            VARCHAR(16) NOT NULL CHECK (protocol IN ('http', 'https', 'tcp', 'udp', 'ssh')),
    host                VARCHAR(255) NOT NULL,                       -- Public hostname or IP
    port                INT NOT NULL CHECK (port BETWEEN 1 AND 65535),
    path                TEXT NULL,                                  -- Optional HTTP(S) URL path
    UNIQUE (deployment_id, name),
    CHECK (path IS NULL OR protocol IN ('http', 'https'))
);

-- Submissions
CREATE TABLE submissions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique submission event identifier
    challenge_id        UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE, -- Challenge attempted
    team_id             UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,      -- Team submitting the flag
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,      -- User submitting the flag
    submitted_flag      VARCHAR(512) NOT NULL,                       -- Raw submitted flag text
    is_correct          BOOLEAN NOT NULL,                            -- Submission verification outcome
    sus                 UUID NULL REFERENCES teams(id),              -- Leaked flag alert: populated with team UUID if dynamic flag was leaked
    ip_address          INET NOT NULL,                               -- Client IP address used during submission
    user_agent          TEXT NULL,                                   -- Client User-Agent string (detects scripts/curl vs browser)
    tls_fingerprint     VARCHAR(64) NULL,                            -- JA3/TLS handshake fingerprint for anti-cheat inspection
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- Submission timestamp for rate limiting and tie-breakers
);

CREATE INDEX idx_submissions_rate_limit ON submissions (team_id, created_at DESC);
CREATE INDEX idx_submissions_challenge ON submissions (challenge_id, is_correct);

-- Solves
CREATE TABLE solves (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique solve entry identifier
    challenge_id        UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE, -- Solved challenge
    team_id             UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,      -- Team credited with the solve
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,      -- User who submitted the successful flag
    points_awarded      INT NOT NULL CHECK (points_awarded >= 0),     -- Score awarded at time of solve (dynamic point snapshot)
    is_first_blood      BOOLEAN DEFAULT FALSE,                       -- Indicates if this solve was first blood
    solved_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW(),       -- Official solve timestamp used for scoreboard ranking
    UNIQUE (challenge_id, team_id)
);

CREATE INDEX idx_solves_scoreboard ON solves (team_id, solved_at);

-- Notifications
CREATE TABLE notifications (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique notification identifier
    title               VARCHAR(255) NOT NULL,                       -- Short headline / alert title
    content             TEXT NOT NULL,                               -- Notification message body in Markdown
    type                VARCHAR(32) NOT NULL DEFAULT 'announcement', -- Category: 'announcement', 'challenge_update', 'first_blood', 'team'
    challenge_id        UUID NULL REFERENCES challenges(id) ON DELETE CASCADE, -- Optional reference for challenge-specific announcements
    team_id             UUID NULL REFERENCES teams(id) ON DELETE CASCADE,      -- NULL = broadcast to all; team UUID for targeted alert
    user_id             UUID NULL REFERENCES users(id) ON DELETE CASCADE,      -- NULL = team/broadcast; user UUID for personal notification
    is_pinned           BOOLEAN DEFAULT FALSE,                       -- Sticky banner toggle for critical announcements
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- Timestamp when notification was issued
);

CREATE INDEX idx_notifications_team ON notifications (team_id, created_at DESC);
CREATE INDEX idx_notifications_user ON notifications (user_id, created_at DESC);

-- User Notification Reads
CREATE TABLE user_notification_reads (
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,      -- User who viewed the notification
    notification_id     UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE, -- Viewed notification
    read_at             TIMESTAMP WITH TIME ZONE DEFAULT NOW(),       -- Timestamp when read
    PRIMARY KEY (user_id, notification_id)
);

-- Competition Config (Singleton row)
CREATE TABLE competition_config (
    id                           INT PRIMARY KEY DEFAULT 1 CHECK (id = 1), -- Singleton row for global competition state
    title                        VARCHAR(255) NOT NULL DEFAULT 'Beast CTF', -- Public competition title
    about                        TEXT NULL,                                -- Short about overview
    description                  TEXT NULL,                                -- Full competition details in Markdown
    rules                        TEXT NULL,                                -- Competition rules and guidelines
    scoring_policy               VARCHAR(32) NOT NULL DEFAULT 'dynamic' CHECK (scoring_policy IN ('dynamic', 'standard')), -- Scoring model
    registration_open            BOOLEAN NOT NULL DEFAULT TRUE,            -- Toggle for new user registration
    starts_at                    TIMESTAMP WITH TIME ZONE NULL,            -- Official competition start time
    ends_at                      TIMESTAMP WITH TIME ZONE NULL,            -- Official competition end time
    is_frozen                    BOOLEAN NOT NULL DEFAULT FALSE,           -- Scoreboard freeze toggle
    frozen_at                    TIMESTAMP WITH TIME ZONE NULL,            -- Timestamp when scoreboard was frozen
    max_team_size                INT NOT NULL DEFAULT 4 CHECK (max_team_size > 0), -- Maximum members per team
    submission_rate_limit_per_min INT NOT NULL DEFAULT 10 CHECK (submission_rate_limit_per_min > 0), -- Flag submission rate limit
    logo_path                    VARCHAR(512) NULL,                        -- Storage path for branding logo
    created_at                   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at                   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Auth Sessions
CREATE TABLE auth_sessions (
    id                  BIGSERIAL PRIMARY KEY,                       -- Monotonic 64-bit integer session identifier
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- User who owns this session
    refresh_token_hash  CHAR(64) NOT NULL UNIQUE,                    -- SHA-256 hash of the rotating refresh token
    user_agent          TEXT NULL,                                   -- Client User-Agent string
    ip_address          INET NOT NULL,                               -- Client IP address at session creation/refresh
    is_revoked          BOOLEAN NOT NULL DEFAULT FALSE,              -- Explicit revocation toggle
    expires_at          TIMESTAMP WITH TIME ZONE NOT NULL,           -- Session expiration timestamp (e.g. 7 days)
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),       -- Session creation timestamp
    last_used_at        TIMESTAMP WITH TIME ZONE NULL                -- Timestamp of last token refresh
);

CREATE INDEX idx_auth_sessions_user ON auth_sessions (user_id, expires_at) WHERE is_revoked IS FALSE;

-- Audit Events
CREATE TABLE audit_events (
    id                  BIGSERIAL PRIMARY KEY,                       -- Monotonic 64-bit integer audit event identifier
    actor_user_id       UUID NULL REFERENCES users(id) ON DELETE SET NULL, -- Admin who triggered the event
    action              VARCHAR(64) NOT NULL,                        -- Event action name (e.g. 'user.ban', 'leaderboard.freeze')
    target_type         VARCHAR(64) NOT NULL,                        -- Target category (e.g. 'user', 'team', 'challenge', 'node')
    target_id           VARCHAR(128) NOT NULL,                       -- Natural key identifier of target entity (never UUID)
    metadata            JSONB NULL,                                  -- Additional structured context (reasons, diffs, options)
    ip_address          INET NOT NULL,                               -- IP address of the actor
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()        -- Timestamp when action occurred
);

CREATE INDEX idx_audit_events_created ON audit_events (created_at DESC);
CREATE INDEX idx_audit_events_actor ON audit_events (actor_user_id, created_at DESC);
CREATE INDEX idx_audit_events_target ON audit_events (target_type, target_id);
