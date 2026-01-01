-- Revert Mesh Networking Migration

DROP TABLE IF EXISTS mesh_connections;
DROP TABLE IF EXISTS mesh_gateway_groups;
DROP TABLE IF EXISTS mesh_gateway_users;
DROP TABLE IF EXISTS mesh_hub_groups;
DROP TABLE IF EXISTS mesh_hub_users;
DROP TABLE IF EXISTS mesh_gateways;
DROP TABLE IF EXISTS mesh_hubs;
