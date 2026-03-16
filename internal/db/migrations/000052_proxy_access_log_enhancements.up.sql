-- Add request/response headers columns (optional, JSONB)
ALTER TABLE proxy_access_logs ADD COLUMN IF NOT EXISTS request_headers JSONB;
ALTER TABLE proxy_access_logs ADD COLUMN IF NOT EXISTS response_headers JSONB;
ALTER TABLE proxy_access_logs ADD COLUMN IF NOT EXISTS response_size_bytes INTEGER;

-- Add logging toggle to proxy_applications
ALTER TABLE proxy_applications ADD COLUMN IF NOT EXISTS access_logging_enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE proxy_applications ADD COLUMN IF NOT EXISTS log_headers BOOLEAN NOT NULL DEFAULT false;
