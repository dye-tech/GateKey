CREATE TABLE session_recordings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_type VARCHAR(20) NOT NULL DEFAULT 'terminal',
    user_id VARCHAR(255) NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    target_node_id VARCHAR(255),
    target_node_name VARCHAR(255),
    storage_path TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL DEFAULT 0,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    recording_format VARCHAR(20) NOT NULL DEFAULT 'asciicast',
    is_complete BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_session_recordings_user ON session_recordings(user_id);
CREATE INDEX idx_session_recordings_created ON session_recordings(created_at DESC);
CREATE INDEX idx_session_recordings_target ON session_recordings(target_node_name);
