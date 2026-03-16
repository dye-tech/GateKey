ALTER TABLE proxy_access_logs DROP COLUMN IF EXISTS request_headers;
ALTER TABLE proxy_access_logs DROP COLUMN IF EXISTS response_headers;
ALTER TABLE proxy_access_logs DROP COLUMN IF EXISTS response_size_bytes;
ALTER TABLE proxy_applications DROP COLUMN IF EXISTS access_logging_enabled;
ALTER TABLE proxy_applications DROP COLUMN IF EXISTS log_headers;
