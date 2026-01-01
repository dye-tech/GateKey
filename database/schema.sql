--
-- PostgreSQL database dump
--


-- Dumped from database version 16.10
-- Dumped by pg_dump version 16.10

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: compute_gateway_config_version(character varying, integer, character varying, cidr, boolean, text); Type: FUNCTION; Schema: public; Owner: gatekey
--

CREATE FUNCTION public.compute_gateway_config_version(p_crypto_profile character varying, p_vpn_port integer, p_vpn_protocol character varying, p_vpn_subnet cidr, p_tls_auth_enabled boolean, p_tls_auth_key text) RETURNS character varying
    LANGUAGE plpgsql IMMUTABLE
    AS $$
BEGIN
    RETURN encode(
        sha256(
            (COALESCE(p_crypto_profile, '') || '|' ||
             COALESCE(p_vpn_port::TEXT, '') || '|' ||
             COALESCE(p_vpn_protocol, '') || '|' ||
             COALESCE(p_vpn_subnet::TEXT, '') || '|' ||
             COALESCE(p_tls_auth_enabled::TEXT, '') || '|' ||
             COALESCE(p_tls_auth_key, ''))::bytea
        ),
        'hex'
    );
END;
$$;


ALTER FUNCTION public.compute_gateway_config_version(p_crypto_profile character varying, p_vpn_port integer, p_vpn_protocol character varying, p_vpn_subnet cidr, p_tls_auth_enabled boolean, p_tls_auth_key text) OWNER TO gatekey;

--
-- Name: update_gateway_config_version(); Type: FUNCTION; Schema: public; Owner: gatekey
--

CREATE FUNCTION public.update_gateway_config_version() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.config_version := compute_gateway_config_version(
        NEW.crypto_profile,
        NEW.vpn_port,
        NEW.vpn_protocol,
        NEW.vpn_subnet,
        NEW.tls_auth_enabled,
        NEW.tls_auth_key
    );
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_gateway_config_version() OWNER TO gatekey;

--
-- Name: update_networks_updated_at(); Type: FUNCTION; Schema: public; Owner: gatekey
--

CREATE FUNCTION public.update_networks_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_networks_updated_at() OWNER TO gatekey;

--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: gatekey
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_updated_at_column() OWNER TO gatekey;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: access_rules; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.access_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    rule_type character varying(50) NOT NULL,
    value character varying(512) NOT NULL,
    port_range character varying(50),
    protocol character varying(20),
    network_id uuid,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT access_rules_rule_type_check CHECK (((rule_type)::text = ANY ((ARRAY['ip'::character varying, 'cidr'::character varying, 'hostname'::character varying, 'hostname_wildcard'::character varying])::text[])))
);


ALTER TABLE public.access_rules OWNER TO gatekey;

--
-- Name: admin_sessions; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.admin_sessions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    token character varying(64) NOT NULL,
    ip_address inet,
    user_agent text DEFAULT ''::text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.admin_sessions OWNER TO gatekey;

--
-- Name: api_keys; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.api_keys (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text DEFAULT ''::text,
    key_hash character varying(64) NOT NULL,
    key_prefix character varying(12) NOT NULL,
    scopes jsonb DEFAULT '[]'::jsonb NOT NULL,
    is_admin_provisioned boolean DEFAULT false,
    provisioned_by uuid,
    expires_at timestamp with time zone,
    last_used_at timestamp with time zone,
    last_used_ip inet,
    is_revoked boolean DEFAULT false,
    revoked_at timestamp with time zone,
    revoked_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    revocation_reason text
);


ALTER TABLE public.api_keys OWNER TO gatekey;

--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.audit_logs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    "timestamp" timestamp with time zone DEFAULT now() NOT NULL,
    event character varying(100) NOT NULL,
    actor_id uuid,
    actor_email character varying(255),
    actor_ip inet NOT NULL,
    resource_type character varying(50) NOT NULL,
    resource_id uuid,
    details jsonb DEFAULT '{}'::jsonb NOT NULL,
    success boolean DEFAULT true NOT NULL
);


ALTER TABLE public.audit_logs OWNER TO gatekey;

--
-- Name: ca_rotation_events; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.ca_rotation_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_id character varying(50) NOT NULL,
    event_type character varying(50) NOT NULL,
    old_fingerprint character varying(64),
    new_fingerprint character varying(64),
    initiated_by character varying(255),
    notes text,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.ca_rotation_events OWNER TO gatekey;

--
-- Name: certificates; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.certificates (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    session_id uuid NOT NULL,
    serial_number character varying(64) NOT NULL,
    subject character varying(255) NOT NULL,
    not_before timestamp with time zone NOT NULL,
    not_after timestamp with time zone NOT NULL,
    fingerprint character varying(64) NOT NULL,
    is_revoked boolean DEFAULT false NOT NULL,
    revoked_at timestamp with time zone,
    revocation_reason character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.certificates OWNER TO gatekey;

--
-- Name: configs; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.configs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    session_id uuid NOT NULL,
    certificate_id uuid NOT NULL,
    gateway_id uuid NOT NULL,
    file_name character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    downloaded_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.configs OWNER TO gatekey;

--
-- Name: connections; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.connections (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    session_id uuid NOT NULL,
    certificate_id uuid NOT NULL,
    gateway_id uuid NOT NULL,
    client_ip inet NOT NULL,
    vpn_ipv4 inet NOT NULL,
    vpn_ipv6 inet,
    bytes_sent bigint DEFAULT 0 NOT NULL,
    bytes_received bigint DEFAULT 0 NOT NULL,
    connected_at timestamp with time zone DEFAULT now() NOT NULL,
    disconnected_at timestamp with time zone,
    disconnect_reason character varying(100)
);


ALTER TABLE public.connections OWNER TO gatekey;

--
-- Name: gateway_networks; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.gateway_networks (
    gateway_id uuid NOT NULL,
    network_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.gateway_networks OWNER TO gatekey;

--
-- Name: gateways; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.gateways (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(100) NOT NULL,
    hostname character varying(255),
    public_ip inet,
    vpn_port integer DEFAULT 1194 NOT NULL,
    vpn_protocol character varying(10) DEFAULT 'udp'::character varying NOT NULL,
    token character varying(64) NOT NULL,
    public_key text,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_active boolean DEFAULT false NOT NULL,
    last_heartbeat timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    crypto_profile character varying(50) DEFAULT 'modern'::character varying NOT NULL,
    vpn_subnet cidr DEFAULT '10.8.0.0/24'::cidr NOT NULL,
    tls_auth_enabled boolean DEFAULT true NOT NULL,
    tls_auth_key text,
    config_version character varying(64),
    full_tunnel_mode boolean DEFAULT false NOT NULL,
    push_dns boolean DEFAULT false NOT NULL,
    dns_servers text[] DEFAULT '{}'::text[] NOT NULL,
    CONSTRAINT chk_gateway_address CHECK (((hostname IS NOT NULL) OR (public_ip IS NOT NULL)))
);


ALTER TABLE public.gateways OWNER TO gatekey;

--
-- Name: COLUMN gateways.crypto_profile; Type: COMMENT; Schema: public; Owner: gatekey
--

COMMENT ON COLUMN public.gateways.crypto_profile IS 'Cryptographic profile: modern (default), fips (FIPS 140-2 compliant), compatible (legacy support)';


--
-- Name: generated_configs; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.generated_configs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id character varying(255) NOT NULL,
    gateway_id uuid NOT NULL,
    gateway_name character varying(255) NOT NULL,
    file_name character varying(255) NOT NULL,
    config_data bytea NOT NULL,
    serial_number character varying(255) NOT NULL,
    fingerprint character varying(255) NOT NULL,
    cli_callback_url character varying(1024),
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    downloaded_at timestamp with time zone,
    auth_token character varying(64) DEFAULT ''::character varying NOT NULL,
    is_revoked boolean DEFAULT false NOT NULL,
    revoked_at timestamp with time zone,
    revoked_reason character varying(255)
);


ALTER TABLE public.generated_configs OWNER TO gatekey;

--
-- Name: group_access_rules; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.group_access_rules (
    group_name character varying(255) NOT NULL,
    access_rule_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.group_access_rules OWNER TO gatekey;

--
-- Name: group_gateways; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.group_gateways (
    group_name character varying(255) NOT NULL,
    gateway_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.group_gateways OWNER TO gatekey;

--
-- Name: group_proxy_applications; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.group_proxy_applications (
    group_name character varying(255) NOT NULL,
    proxy_app_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.group_proxy_applications OWNER TO gatekey;

--
-- Name: local_users; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.local_users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    username character varying(100) NOT NULL,
    password_hash text NOT NULL,
    email character varying(255) NOT NULL,
    is_admin boolean DEFAULT false NOT NULL,
    last_login_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.local_users OWNER TO gatekey;

--
-- Name: login_logs; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.login_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id character varying(255) NOT NULL,
    user_email character varying(255) NOT NULL,
    user_name character varying(255),
    provider character varying(50) NOT NULL,
    provider_name character varying(100),
    ip_address inet NOT NULL,
    user_agent text,
    country character varying(100),
    city character varying(100),
    success boolean DEFAULT true NOT NULL,
    failure_reason character varying(255),
    session_id character varying(255),
    created_at timestamp with time zone DEFAULT now(),
    country_code character varying(2)
);


ALTER TABLE public.login_logs OWNER TO gatekey;

--
-- Name: mesh_connections; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.mesh_connections (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    hub_id uuid NOT NULL,
    user_id uuid NOT NULL,
    client_ip inet NOT NULL,
    tunnel_ip inet NOT NULL,
    bytes_sent bigint DEFAULT 0 NOT NULL,
    bytes_received bigint DEFAULT 0 NOT NULL,
    connected_at timestamp with time zone DEFAULT now() NOT NULL,
    disconnected_at timestamp with time zone,
    disconnect_reason character varying(100)
);


ALTER TABLE public.mesh_connections OWNER TO gatekey;

--
-- Name: mesh_gateway_groups; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.mesh_gateway_groups (
    gateway_id uuid NOT NULL,
    group_name character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.mesh_gateway_groups OWNER TO gatekey;

--
-- Name: mesh_gateway_users; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.mesh_gateway_users (
    gateway_id uuid NOT NULL,
    user_id character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.mesh_gateway_users OWNER TO gatekey;

--
-- Name: mesh_gateways; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.mesh_gateways (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    hub_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    local_networks text[] DEFAULT '{}'::text[] NOT NULL,
    tunnel_ip inet,
    client_cert text,
    client_key text,
    token character varying(64) NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    status_message text,
    last_seen timestamp with time zone,
    bytes_sent bigint DEFAULT 0 NOT NULL,
    bytes_received bigint DEFAULT 0 NOT NULL,
    remote_ip inet,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    full_tunnel_mode boolean DEFAULT false NOT NULL,
    push_dns boolean DEFAULT false NOT NULL,
    dns_servers text[] DEFAULT '{}'::text[] NOT NULL
);


ALTER TABLE public.mesh_gateways OWNER TO gatekey;

--
-- Name: mesh_hub_groups; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.mesh_hub_groups (
    hub_id uuid NOT NULL,
    group_name character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.mesh_hub_groups OWNER TO gatekey;

--
-- Name: mesh_hub_networks; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.mesh_hub_networks (
    hub_id uuid NOT NULL,
    network_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.mesh_hub_networks OWNER TO gatekey;

--
-- Name: mesh_hub_users; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.mesh_hub_users (
    hub_id uuid NOT NULL,
    user_id character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.mesh_hub_users OWNER TO gatekey;

--
-- Name: mesh_hubs; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.mesh_hubs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(100) NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    public_endpoint character varying(255) NOT NULL,
    vpn_port integer DEFAULT 1194 NOT NULL,
    vpn_protocol character varying(10) DEFAULT 'udp'::character varying NOT NULL,
    vpn_subnet cidr DEFAULT '172.30.0.0/16'::cidr NOT NULL,
    crypto_profile character varying(50) DEFAULT 'fips'::character varying NOT NULL,
    tls_auth_enabled boolean DEFAULT true NOT NULL,
    tls_auth_key text,
    ca_cert text,
    ca_key text,
    server_cert text,
    server_key text,
    dh_params text,
    api_token character varying(64) NOT NULL,
    control_plane_url text NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    status_message text,
    last_heartbeat timestamp with time zone,
    connected_gateways integer DEFAULT 0 NOT NULL,
    connected_clients integer DEFAULT 0 NOT NULL,
    config_version character varying(64),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    local_networks text[] DEFAULT '{}'::text[] NOT NULL,
    full_tunnel_mode boolean DEFAULT false NOT NULL,
    push_dns boolean DEFAULT false NOT NULL,
    dns_servers text[] DEFAULT '{}'::text[] NOT NULL
);


ALTER TABLE public.mesh_hubs OWNER TO gatekey;

--
-- Name: networks; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.networks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    cidr cidr NOT NULL,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.networks OWNER TO gatekey;

--
-- Name: oauth_states; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.oauth_states (
    state character varying(255) NOT NULL,
    provider character varying(255) NOT NULL,
    provider_type character varying(50) NOT NULL,
    nonce character varying(255),
    relay_state character varying(255),
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    cli_callback_url text
);


ALTER TABLE public.oauth_states OWNER TO gatekey;

--
-- Name: oidc_providers; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.oidc_providers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(100) NOT NULL,
    display_name character varying(255) NOT NULL,
    issuer text NOT NULL,
    client_id character varying(255) NOT NULL,
    client_secret text NOT NULL,
    redirect_url text NOT NULL,
    scopes jsonb DEFAULT '["openid", "profile", "email"]'::jsonb NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    admin_group character varying(255) DEFAULT NULL::character varying
);


ALTER TABLE public.oidc_providers OWNER TO gatekey;

--
-- Name: pki_ca; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.pki_ca (
    id character varying(50) DEFAULT 'default'::character varying NOT NULL,
    certificate_pem text NOT NULL,
    private_key_pem text NOT NULL,
    serial_number character varying(100),
    not_before timestamp with time zone NOT NULL,
    not_after timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    status character varying(20) DEFAULT 'active'::character varying,
    fingerprint character varying(64),
    description text
);


ALTER TABLE public.pki_ca OWNER TO gatekey;

--
-- Name: policies; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.policies (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(100) NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    priority integer DEFAULT 100 NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by uuid
);


ALTER TABLE public.policies OWNER TO gatekey;

--
-- Name: policy_rules; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.policy_rules (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    policy_id uuid NOT NULL,
    action character varying(10) NOT NULL,
    subject jsonb DEFAULT '{}'::jsonb NOT NULL,
    resource jsonb DEFAULT '{}'::jsonb NOT NULL,
    conditions jsonb DEFAULT '{}'::jsonb NOT NULL,
    priority integer DEFAULT 100 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT policy_rules_action_check CHECK (((action)::text = ANY ((ARRAY['allow'::character varying, 'deny'::character varying])::text[])))
);


ALTER TABLE public.policy_rules OWNER TO gatekey;

--
-- Name: proxy_access_logs; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.proxy_access_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    proxy_app_id uuid NOT NULL,
    user_id character varying(255) NOT NULL,
    user_email character varying(255),
    request_method character varying(10) NOT NULL,
    request_path text NOT NULL,
    response_status integer,
    response_time_ms integer,
    client_ip inet,
    user_agent text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.proxy_access_logs OWNER TO gatekey;

--
-- Name: proxy_applications; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.proxy_applications (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(100) NOT NULL,
    description text,
    internal_url text NOT NULL,
    icon_url text,
    is_active boolean DEFAULT true,
    preserve_host_header boolean DEFAULT false,
    strip_prefix boolean DEFAULT true,
    inject_headers jsonb DEFAULT '{}'::jsonb,
    allowed_headers jsonb DEFAULT '["*"]'::jsonb,
    websocket_enabled boolean DEFAULT true,
    timeout_seconds integer DEFAULT 30,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.proxy_applications OWNER TO gatekey;

--
-- Name: saml_providers; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.saml_providers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(100) NOT NULL,
    display_name character varying(255) NOT NULL,
    idp_metadata_url text NOT NULL,
    entity_id text NOT NULL,
    acs_url text NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    admin_group character varying(255) DEFAULT NULL::character varying
);


ALTER TABLE public.saml_providers OWNER TO gatekey;

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO gatekey;

--
-- Name: sessions; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.sessions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    token character varying(64) NOT NULL,
    ip_address inet NOT NULL,
    user_agent text DEFAULT ''::text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone
);


ALTER TABLE public.sessions OWNER TO gatekey;

--
-- Name: sso_sessions; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.sso_sessions (
    token character varying(255) NOT NULL,
    user_id character varying(255) NOT NULL,
    username character varying(255),
    email character varying(255),
    name character varying(255),
    groups jsonb DEFAULT '[]'::jsonb,
    provider character varying(255),
    is_admin boolean DEFAULT false,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.sso_sessions OWNER TO gatekey;

--
-- Name: system_settings; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.system_settings (
    key character varying(255) NOT NULL,
    value text NOT NULL,
    description text,
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.system_settings OWNER TO gatekey;

--
-- Name: user_access_rules; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.user_access_rules (
    user_id uuid NOT NULL,
    access_rule_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.user_access_rules OWNER TO gatekey;

--
-- Name: user_gateways; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.user_gateways (
    user_id character varying(255) NOT NULL,
    gateway_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.user_gateways OWNER TO gatekey;

--
-- Name: user_proxy_applications; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.user_proxy_applications (
    user_id character varying(255) NOT NULL,
    proxy_app_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.user_proxy_applications OWNER TO gatekey;

--
-- Name: users; Type: TABLE; Schema: public; Owner: gatekey
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    external_id character varying(255) NOT NULL,
    provider character varying(100) NOT NULL,
    email character varying(255) NOT NULL,
    name character varying(255) DEFAULT ''::character varying NOT NULL,
    groups jsonb DEFAULT '[]'::jsonb NOT NULL,
    attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_admin boolean DEFAULT false NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    last_login_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.users OWNER TO gatekey;

--
-- Name: access_rules access_rules_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.access_rules
    ADD CONSTRAINT access_rules_pkey PRIMARY KEY (id);


--
-- Name: admin_sessions admin_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.admin_sessions
    ADD CONSTRAINT admin_sessions_pkey PRIMARY KEY (id);


--
-- Name: admin_sessions admin_sessions_token_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.admin_sessions
    ADD CONSTRAINT admin_sessions_token_key UNIQUE (token);


--
-- Name: api_keys api_keys_key_hash_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_key_hash_key UNIQUE (key_hash);


--
-- Name: api_keys api_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);


--
-- Name: ca_rotation_events ca_rotation_events_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.ca_rotation_events
    ADD CONSTRAINT ca_rotation_events_pkey PRIMARY KEY (id);


--
-- Name: certificates certificates_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.certificates
    ADD CONSTRAINT certificates_pkey PRIMARY KEY (id);


--
-- Name: certificates certificates_serial_number_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.certificates
    ADD CONSTRAINT certificates_serial_number_key UNIQUE (serial_number);


--
-- Name: configs configs_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.configs
    ADD CONSTRAINT configs_pkey PRIMARY KEY (id);


--
-- Name: connections connections_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.connections
    ADD CONSTRAINT connections_pkey PRIMARY KEY (id);


--
-- Name: gateway_networks gateway_networks_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.gateway_networks
    ADD CONSTRAINT gateway_networks_pkey PRIMARY KEY (gateway_id, network_id);


--
-- Name: gateways gateways_name_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.gateways
    ADD CONSTRAINT gateways_name_key UNIQUE (name);


--
-- Name: gateways gateways_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.gateways
    ADD CONSTRAINT gateways_pkey PRIMARY KEY (id);


--
-- Name: generated_configs generated_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.generated_configs
    ADD CONSTRAINT generated_configs_pkey PRIMARY KEY (id);


--
-- Name: group_access_rules group_access_rules_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.group_access_rules
    ADD CONSTRAINT group_access_rules_pkey PRIMARY KEY (group_name, access_rule_id);


--
-- Name: group_gateways group_gateways_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.group_gateways
    ADD CONSTRAINT group_gateways_pkey PRIMARY KEY (group_name, gateway_id);


--
-- Name: group_proxy_applications group_proxy_applications_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.group_proxy_applications
    ADD CONSTRAINT group_proxy_applications_pkey PRIMARY KEY (group_name, proxy_app_id);


--
-- Name: local_users local_users_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.local_users
    ADD CONSTRAINT local_users_pkey PRIMARY KEY (id);


--
-- Name: local_users local_users_username_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.local_users
    ADD CONSTRAINT local_users_username_key UNIQUE (username);


--
-- Name: login_logs login_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.login_logs
    ADD CONSTRAINT login_logs_pkey PRIMARY KEY (id);


--
-- Name: mesh_connections mesh_connections_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_connections
    ADD CONSTRAINT mesh_connections_pkey PRIMARY KEY (id);


--
-- Name: mesh_gateway_groups mesh_gateway_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_gateway_groups
    ADD CONSTRAINT mesh_gateway_groups_pkey PRIMARY KEY (gateway_id, group_name);


--
-- Name: mesh_gateway_users mesh_gateway_users_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_gateway_users
    ADD CONSTRAINT mesh_gateway_users_pkey PRIMARY KEY (gateway_id, user_id);


--
-- Name: mesh_gateways mesh_gateways_hub_id_name_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_gateways
    ADD CONSTRAINT mesh_gateways_hub_id_name_key UNIQUE (hub_id, name);


--
-- Name: mesh_gateways mesh_gateways_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_gateways
    ADD CONSTRAINT mesh_gateways_pkey PRIMARY KEY (id);


--
-- Name: mesh_hub_groups mesh_hub_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hub_groups
    ADD CONSTRAINT mesh_hub_groups_pkey PRIMARY KEY (hub_id, group_name);


--
-- Name: mesh_hub_networks mesh_hub_networks_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hub_networks
    ADD CONSTRAINT mesh_hub_networks_pkey PRIMARY KEY (hub_id, network_id);


--
-- Name: mesh_hub_users mesh_hub_users_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hub_users
    ADD CONSTRAINT mesh_hub_users_pkey PRIMARY KEY (hub_id, user_id);


--
-- Name: mesh_hubs mesh_hubs_name_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hubs
    ADD CONSTRAINT mesh_hubs_name_key UNIQUE (name);


--
-- Name: mesh_hubs mesh_hubs_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hubs
    ADD CONSTRAINT mesh_hubs_pkey PRIMARY KEY (id);


--
-- Name: networks networks_name_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.networks
    ADD CONSTRAINT networks_name_key UNIQUE (name);


--
-- Name: networks networks_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.networks
    ADD CONSTRAINT networks_pkey PRIMARY KEY (id);


--
-- Name: oauth_states oauth_states_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.oauth_states
    ADD CONSTRAINT oauth_states_pkey PRIMARY KEY (state);


--
-- Name: oidc_providers oidc_providers_name_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.oidc_providers
    ADD CONSTRAINT oidc_providers_name_key UNIQUE (name);


--
-- Name: oidc_providers oidc_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.oidc_providers
    ADD CONSTRAINT oidc_providers_pkey PRIMARY KEY (id);


--
-- Name: pki_ca pki_ca_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.pki_ca
    ADD CONSTRAINT pki_ca_pkey PRIMARY KEY (id);


--
-- Name: policies policies_name_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.policies
    ADD CONSTRAINT policies_name_key UNIQUE (name);


--
-- Name: policies policies_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.policies
    ADD CONSTRAINT policies_pkey PRIMARY KEY (id);


--
-- Name: policy_rules policy_rules_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.policy_rules
    ADD CONSTRAINT policy_rules_pkey PRIMARY KEY (id);


--
-- Name: proxy_access_logs proxy_access_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.proxy_access_logs
    ADD CONSTRAINT proxy_access_logs_pkey PRIMARY KEY (id);


--
-- Name: proxy_applications proxy_applications_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.proxy_applications
    ADD CONSTRAINT proxy_applications_pkey PRIMARY KEY (id);


--
-- Name: proxy_applications proxy_applications_slug_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.proxy_applications
    ADD CONSTRAINT proxy_applications_slug_key UNIQUE (slug);


--
-- Name: saml_providers saml_providers_name_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.saml_providers
    ADD CONSTRAINT saml_providers_name_key UNIQUE (name);


--
-- Name: saml_providers saml_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.saml_providers
    ADD CONSTRAINT saml_providers_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: sessions sessions_token_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_token_key UNIQUE (token);


--
-- Name: sso_sessions sso_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.sso_sessions
    ADD CONSTRAINT sso_sessions_pkey PRIMARY KEY (token);


--
-- Name: system_settings system_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.system_settings
    ADD CONSTRAINT system_settings_pkey PRIMARY KEY (key);


--
-- Name: user_access_rules user_access_rules_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.user_access_rules
    ADD CONSTRAINT user_access_rules_pkey PRIMARY KEY (user_id, access_rule_id);


--
-- Name: user_gateways user_gateways_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.user_gateways
    ADD CONSTRAINT user_gateways_pkey PRIMARY KEY (user_id, gateway_id);


--
-- Name: user_proxy_applications user_proxy_applications_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.user_proxy_applications
    ADD CONSTRAINT user_proxy_applications_pkey PRIMARY KEY (user_id, proxy_app_id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_provider_external_id_key; Type: CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_provider_external_id_key UNIQUE (provider, external_id);


--
-- Name: idx_access_rules_network; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_access_rules_network ON public.access_rules USING btree (network_id);


--
-- Name: idx_access_rules_type; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_access_rules_type ON public.access_rules USING btree (rule_type);


--
-- Name: idx_admin_sessions_expires_at; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_admin_sessions_expires_at ON public.admin_sessions USING btree (expires_at);


--
-- Name: idx_admin_sessions_token; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_admin_sessions_token ON public.admin_sessions USING btree (token);


--
-- Name: idx_admin_sessions_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_admin_sessions_user_id ON public.admin_sessions USING btree (user_id);


--
-- Name: idx_api_keys_is_revoked; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_api_keys_is_revoked ON public.api_keys USING btree (is_revoked);


--
-- Name: idx_api_keys_key_hash; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_api_keys_key_hash ON public.api_keys USING btree (key_hash);


--
-- Name: idx_api_keys_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_api_keys_user_id ON public.api_keys USING btree (user_id);


--
-- Name: idx_audit_logs_actor_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_audit_logs_actor_id ON public.audit_logs USING btree (actor_id);


--
-- Name: idx_audit_logs_event; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_audit_logs_event ON public.audit_logs USING btree (event);


--
-- Name: idx_audit_logs_resource; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_audit_logs_resource ON public.audit_logs USING btree (resource_type, resource_id);


--
-- Name: idx_audit_logs_timestamp; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_audit_logs_timestamp ON public.audit_logs USING btree ("timestamp");


--
-- Name: idx_ca_rotation_events_created; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_ca_rotation_events_created ON public.ca_rotation_events USING btree (created_at DESC);


--
-- Name: idx_certificates_fingerprint; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_certificates_fingerprint ON public.certificates USING btree (fingerprint);


--
-- Name: idx_certificates_is_revoked; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_certificates_is_revoked ON public.certificates USING btree (is_revoked);


--
-- Name: idx_certificates_not_after; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_certificates_not_after ON public.certificates USING btree (not_after);


--
-- Name: idx_certificates_serial_number; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_certificates_serial_number ON public.certificates USING btree (serial_number);


--
-- Name: idx_certificates_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_certificates_user_id ON public.certificates USING btree (user_id);


--
-- Name: idx_configs_expires_at; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_configs_expires_at ON public.configs USING btree (expires_at);


--
-- Name: idx_configs_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_configs_user_id ON public.configs USING btree (user_id);


--
-- Name: idx_connections_active; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_connections_active ON public.connections USING btree (disconnected_at) WHERE (disconnected_at IS NULL);


--
-- Name: idx_connections_connected_at; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_connections_connected_at ON public.connections USING btree (connected_at);


--
-- Name: idx_connections_gateway_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_connections_gateway_id ON public.connections USING btree (gateway_id);


--
-- Name: idx_connections_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_connections_user_id ON public.connections USING btree (user_id);


--
-- Name: idx_gateway_networks_gateway; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_gateway_networks_gateway ON public.gateway_networks USING btree (gateway_id);


--
-- Name: idx_gateway_networks_network; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_gateway_networks_network ON public.gateway_networks USING btree (network_id);


--
-- Name: idx_gateways_is_active; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_gateways_is_active ON public.gateways USING btree (is_active);


--
-- Name: idx_gateways_name; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_gateways_name ON public.gateways USING btree (name);


--
-- Name: idx_generated_configs_active; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_generated_configs_active ON public.generated_configs USING btree (user_id, is_revoked) WHERE (is_revoked = false);


--
-- Name: idx_generated_configs_auth_token; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_generated_configs_auth_token ON public.generated_configs USING btree (auth_token) WHERE ((auth_token)::text <> ''::text);


--
-- Name: idx_generated_configs_expires_at; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_generated_configs_expires_at ON public.generated_configs USING btree (expires_at);


--
-- Name: idx_generated_configs_gateway_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_generated_configs_gateway_id ON public.generated_configs USING btree (gateway_id);


--
-- Name: idx_generated_configs_serial_number; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_generated_configs_serial_number ON public.generated_configs USING btree (serial_number);


--
-- Name: idx_generated_configs_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_generated_configs_user_id ON public.generated_configs USING btree (user_id);


--
-- Name: idx_group_access_rules_group; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_group_access_rules_group ON public.group_access_rules USING btree (group_name);


--
-- Name: idx_group_gateways_gateway; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_group_gateways_gateway ON public.group_gateways USING btree (gateway_id);


--
-- Name: idx_group_gateways_group; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_group_gateways_group ON public.group_gateways USING btree (group_name);


--
-- Name: idx_group_proxy_apps_app; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_group_proxy_apps_app ON public.group_proxy_applications USING btree (proxy_app_id);


--
-- Name: idx_group_proxy_apps_group; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_group_proxy_apps_group ON public.group_proxy_applications USING btree (group_name);


--
-- Name: idx_local_users_username; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_local_users_username ON public.local_users USING btree (username);


--
-- Name: idx_login_logs_created_at; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_login_logs_created_at ON public.login_logs USING btree (created_at DESC);


--
-- Name: idx_login_logs_ip_address; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_login_logs_ip_address ON public.login_logs USING btree (ip_address);


--
-- Name: idx_login_logs_success; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_login_logs_success ON public.login_logs USING btree (success);


--
-- Name: idx_login_logs_user_email; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_login_logs_user_email ON public.login_logs USING btree (user_email);


--
-- Name: idx_login_logs_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_login_logs_user_id ON public.login_logs USING btree (user_id);


--
-- Name: idx_mesh_connections_active; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_connections_active ON public.mesh_connections USING btree (disconnected_at) WHERE (disconnected_at IS NULL);


--
-- Name: idx_mesh_connections_hub_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_connections_hub_id ON public.mesh_connections USING btree (hub_id);


--
-- Name: idx_mesh_connections_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_connections_user_id ON public.mesh_connections USING btree (user_id);


--
-- Name: idx_mesh_gateways_hub_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_gateways_hub_id ON public.mesh_gateways USING btree (hub_id);


--
-- Name: idx_mesh_gateways_status; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_gateways_status ON public.mesh_gateways USING btree (status);


--
-- Name: idx_mesh_gateways_token; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_gateways_token ON public.mesh_gateways USING btree (token);


--
-- Name: idx_mesh_hub_networks_hub; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_hub_networks_hub ON public.mesh_hub_networks USING btree (hub_id);


--
-- Name: idx_mesh_hub_networks_network; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_hub_networks_network ON public.mesh_hub_networks USING btree (network_id);


--
-- Name: idx_mesh_hubs_name; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_hubs_name ON public.mesh_hubs USING btree (name);


--
-- Name: idx_mesh_hubs_status; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_mesh_hubs_status ON public.mesh_hubs USING btree (status);


--
-- Name: idx_networks_cidr; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_networks_cidr ON public.networks USING gist (cidr inet_ops);


--
-- Name: idx_oauth_states_expires; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_oauth_states_expires ON public.oauth_states USING btree (expires_at);


--
-- Name: idx_oidc_providers_enabled; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_oidc_providers_enabled ON public.oidc_providers USING btree (is_enabled);


--
-- Name: idx_oidc_providers_name; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_oidc_providers_name ON public.oidc_providers USING btree (name);


--
-- Name: idx_policies_is_enabled; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_policies_is_enabled ON public.policies USING btree (is_enabled);


--
-- Name: idx_policies_priority; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_policies_priority ON public.policies USING btree (priority);


--
-- Name: idx_policy_rules_policy_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_policy_rules_policy_id ON public.policy_rules USING btree (policy_id);


--
-- Name: idx_policy_rules_priority; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_policy_rules_priority ON public.policy_rules USING btree (priority);


--
-- Name: idx_proxy_access_logs_app; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_proxy_access_logs_app ON public.proxy_access_logs USING btree (proxy_app_id);


--
-- Name: idx_proxy_access_logs_time; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_proxy_access_logs_time ON public.proxy_access_logs USING btree (created_at);


--
-- Name: idx_proxy_access_logs_user; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_proxy_access_logs_user ON public.proxy_access_logs USING btree (user_id);


--
-- Name: idx_proxy_applications_active; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_proxy_applications_active ON public.proxy_applications USING btree (is_active) WHERE (is_active = true);


--
-- Name: idx_proxy_applications_slug; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_proxy_applications_slug ON public.proxy_applications USING btree (slug);


--
-- Name: idx_saml_providers_enabled; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_saml_providers_enabled ON public.saml_providers USING btree (is_enabled);


--
-- Name: idx_saml_providers_name; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_saml_providers_name ON public.saml_providers USING btree (name);


--
-- Name: idx_sessions_expires_at; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_sessions_expires_at ON public.sessions USING btree (expires_at);


--
-- Name: idx_sessions_token; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_sessions_token ON public.sessions USING btree (token);


--
-- Name: idx_sessions_user_id; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_sessions_user_id ON public.sessions USING btree (user_id);


--
-- Name: idx_sso_sessions_expires; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_sso_sessions_expires ON public.sso_sessions USING btree (expires_at);


--
-- Name: idx_sso_sessions_user; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_sso_sessions_user ON public.sso_sessions USING btree (user_id);


--
-- Name: idx_user_access_rules_user; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_user_access_rules_user ON public.user_access_rules USING btree (user_id);


--
-- Name: idx_user_gateways_gateway; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_user_gateways_gateway ON public.user_gateways USING btree (gateway_id);


--
-- Name: idx_user_gateways_user; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_user_gateways_user ON public.user_gateways USING btree (user_id);


--
-- Name: idx_user_proxy_apps_app; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_user_proxy_apps_app ON public.user_proxy_applications USING btree (proxy_app_id);


--
-- Name: idx_user_proxy_apps_user; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_user_proxy_apps_user ON public.user_proxy_applications USING btree (user_id);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_users_groups; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_users_groups ON public.users USING gin (groups);


--
-- Name: idx_users_provider; Type: INDEX; Schema: public; Owner: gatekey
--

CREATE INDEX idx_users_provider ON public.users USING btree (provider);


--
-- Name: access_rules access_rules_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER access_rules_updated_at BEFORE UPDATE ON public.access_rules FOR EACH ROW EXECUTE FUNCTION public.update_networks_updated_at();


--
-- Name: networks networks_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER networks_updated_at BEFORE UPDATE ON public.networks FOR EACH ROW EXECUTE FUNCTION public.update_networks_updated_at();


--
-- Name: proxy_applications proxy_applications_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER proxy_applications_updated_at BEFORE UPDATE ON public.proxy_applications FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: gateways trigger_gateway_config_version; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER trigger_gateway_config_version BEFORE INSERT OR UPDATE ON public.gateways FOR EACH ROW EXECUTE FUNCTION public.update_gateway_config_version();


--
-- Name: gateways update_gateways_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER update_gateways_updated_at BEFORE UPDATE ON public.gateways FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: local_users update_local_users_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER update_local_users_updated_at BEFORE UPDATE ON public.local_users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mesh_gateways update_mesh_gateways_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER update_mesh_gateways_updated_at BEFORE UPDATE ON public.mesh_gateways FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: mesh_hubs update_mesh_hubs_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER update_mesh_hubs_updated_at BEFORE UPDATE ON public.mesh_hubs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: oidc_providers update_oidc_providers_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER update_oidc_providers_updated_at BEFORE UPDATE ON public.oidc_providers FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: policies update_policies_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON public.policies FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: saml_providers update_saml_providers_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER update_saml_providers_updated_at BEFORE UPDATE ON public.saml_providers FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: users update_users_updated_at; Type: TRIGGER; Schema: public; Owner: gatekey
--

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: access_rules access_rules_network_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.access_rules
    ADD CONSTRAINT access_rules_network_id_fkey FOREIGN KEY (network_id) REFERENCES public.networks(id) ON DELETE CASCADE;


--
-- Name: admin_sessions admin_sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.admin_sessions
    ADD CONSTRAINT admin_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.local_users(id) ON DELETE CASCADE;


--
-- Name: api_keys api_keys_provisioned_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_provisioned_by_fkey FOREIGN KEY (provisioned_by) REFERENCES public.users(id);


--
-- Name: api_keys api_keys_revoked_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_revoked_by_fkey FOREIGN KEY (revoked_by) REFERENCES public.users(id);


--
-- Name: api_keys api_keys_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: audit_logs audit_logs_actor_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: certificates certificates_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.certificates
    ADD CONSTRAINT certificates_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.sessions(id) ON DELETE CASCADE;


--
-- Name: certificates certificates_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.certificates
    ADD CONSTRAINT certificates_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: configs configs_certificate_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.configs
    ADD CONSTRAINT configs_certificate_id_fkey FOREIGN KEY (certificate_id) REFERENCES public.certificates(id) ON DELETE CASCADE;


--
-- Name: configs configs_gateway_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.configs
    ADD CONSTRAINT configs_gateway_id_fkey FOREIGN KEY (gateway_id) REFERENCES public.gateways(id) ON DELETE CASCADE;


--
-- Name: configs configs_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.configs
    ADD CONSTRAINT configs_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.sessions(id) ON DELETE CASCADE;


--
-- Name: configs configs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.configs
    ADD CONSTRAINT configs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: connections connections_certificate_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.connections
    ADD CONSTRAINT connections_certificate_id_fkey FOREIGN KEY (certificate_id) REFERENCES public.certificates(id) ON DELETE CASCADE;


--
-- Name: connections connections_gateway_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.connections
    ADD CONSTRAINT connections_gateway_id_fkey FOREIGN KEY (gateway_id) REFERENCES public.gateways(id) ON DELETE CASCADE;


--
-- Name: connections connections_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.connections
    ADD CONSTRAINT connections_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.sessions(id) ON DELETE CASCADE;


--
-- Name: connections connections_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.connections
    ADD CONSTRAINT connections_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: gateway_networks gateway_networks_gateway_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.gateway_networks
    ADD CONSTRAINT gateway_networks_gateway_id_fkey FOREIGN KEY (gateway_id) REFERENCES public.gateways(id) ON DELETE CASCADE;


--
-- Name: gateway_networks gateway_networks_network_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.gateway_networks
    ADD CONSTRAINT gateway_networks_network_id_fkey FOREIGN KEY (network_id) REFERENCES public.networks(id) ON DELETE CASCADE;


--
-- Name: generated_configs generated_configs_gateway_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.generated_configs
    ADD CONSTRAINT generated_configs_gateway_id_fkey FOREIGN KEY (gateway_id) REFERENCES public.gateways(id) ON DELETE CASCADE;


--
-- Name: group_access_rules group_access_rules_access_rule_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.group_access_rules
    ADD CONSTRAINT group_access_rules_access_rule_id_fkey FOREIGN KEY (access_rule_id) REFERENCES public.access_rules(id) ON DELETE CASCADE;


--
-- Name: group_gateways group_gateways_gateway_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.group_gateways
    ADD CONSTRAINT group_gateways_gateway_id_fkey FOREIGN KEY (gateway_id) REFERENCES public.gateways(id) ON DELETE CASCADE;


--
-- Name: group_proxy_applications group_proxy_applications_proxy_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.group_proxy_applications
    ADD CONSTRAINT group_proxy_applications_proxy_app_id_fkey FOREIGN KEY (proxy_app_id) REFERENCES public.proxy_applications(id) ON DELETE CASCADE;


--
-- Name: mesh_connections mesh_connections_hub_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_connections
    ADD CONSTRAINT mesh_connections_hub_id_fkey FOREIGN KEY (hub_id) REFERENCES public.mesh_hubs(id) ON DELETE CASCADE;


--
-- Name: mesh_connections mesh_connections_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_connections
    ADD CONSTRAINT mesh_connections_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: mesh_gateway_groups mesh_gateway_groups_gateway_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_gateway_groups
    ADD CONSTRAINT mesh_gateway_groups_gateway_id_fkey FOREIGN KEY (gateway_id) REFERENCES public.mesh_gateways(id) ON DELETE CASCADE;


--
-- Name: mesh_gateway_users mesh_gateway_users_gateway_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_gateway_users
    ADD CONSTRAINT mesh_gateway_users_gateway_id_fkey FOREIGN KEY (gateway_id) REFERENCES public.mesh_gateways(id) ON DELETE CASCADE;


--
-- Name: mesh_gateways mesh_gateways_hub_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_gateways
    ADD CONSTRAINT mesh_gateways_hub_id_fkey FOREIGN KEY (hub_id) REFERENCES public.mesh_hubs(id) ON DELETE CASCADE;


--
-- Name: mesh_hub_groups mesh_hub_groups_hub_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hub_groups
    ADD CONSTRAINT mesh_hub_groups_hub_id_fkey FOREIGN KEY (hub_id) REFERENCES public.mesh_hubs(id) ON DELETE CASCADE;


--
-- Name: mesh_hub_networks mesh_hub_networks_hub_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hub_networks
    ADD CONSTRAINT mesh_hub_networks_hub_id_fkey FOREIGN KEY (hub_id) REFERENCES public.mesh_hubs(id) ON DELETE CASCADE;


--
-- Name: mesh_hub_networks mesh_hub_networks_network_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hub_networks
    ADD CONSTRAINT mesh_hub_networks_network_id_fkey FOREIGN KEY (network_id) REFERENCES public.networks(id) ON DELETE CASCADE;


--
-- Name: mesh_hub_users mesh_hub_users_hub_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.mesh_hub_users
    ADD CONSTRAINT mesh_hub_users_hub_id_fkey FOREIGN KEY (hub_id) REFERENCES public.mesh_hubs(id) ON DELETE CASCADE;


--
-- Name: policies policies_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.policies
    ADD CONSTRAINT policies_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: policy_rules policy_rules_policy_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.policy_rules
    ADD CONSTRAINT policy_rules_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES public.policies(id) ON DELETE CASCADE;


--
-- Name: proxy_access_logs proxy_access_logs_proxy_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.proxy_access_logs
    ADD CONSTRAINT proxy_access_logs_proxy_app_id_fkey FOREIGN KEY (proxy_app_id) REFERENCES public.proxy_applications(id) ON DELETE CASCADE;


--
-- Name: sessions sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_access_rules user_access_rules_access_rule_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.user_access_rules
    ADD CONSTRAINT user_access_rules_access_rule_id_fkey FOREIGN KEY (access_rule_id) REFERENCES public.access_rules(id) ON DELETE CASCADE;


--
-- Name: user_access_rules user_access_rules_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.user_access_rules
    ADD CONSTRAINT user_access_rules_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_gateways user_gateways_gateway_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.user_gateways
    ADD CONSTRAINT user_gateways_gateway_id_fkey FOREIGN KEY (gateway_id) REFERENCES public.gateways(id) ON DELETE CASCADE;


--
-- Name: user_proxy_applications user_proxy_applications_proxy_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: gatekey
--

ALTER TABLE ONLY public.user_proxy_applications
    ADD CONSTRAINT user_proxy_applications_proxy_app_id_fkey FOREIGN KEY (proxy_app_id) REFERENCES public.proxy_applications(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--


