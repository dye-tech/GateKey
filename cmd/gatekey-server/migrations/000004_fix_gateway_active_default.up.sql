-- Fix: New gateways should default to inactive until they connect
-- A gateway is only "active" when it has sent a heartbeat

-- Change the default for new gateways
ALTER TABLE gateways ALTER COLUMN is_active SET DEFAULT FALSE;

-- Mark existing gateways as inactive if they have never sent a heartbeat
UPDATE gateways SET is_active = false WHERE last_heartbeat IS NULL;

-- Mark gateways as inactive if they haven't sent a heartbeat in the last 2 minutes
UPDATE gateways SET is_active = false WHERE last_heartbeat < NOW() - INTERVAL '2 minutes';
