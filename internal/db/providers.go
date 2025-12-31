package db

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrProviderExists   = errors.New("provider already exists")
)

// OIDCProvider represents an OIDC provider configuration
type OIDCProvider struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Issuer       string   `json:"issuer"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
	AdminGroup   string   `json:"admin_group,omitempty"`
	Enabled      bool     `json:"enabled"`
}

// SAMLProvider represents a SAML provider configuration
type SAMLProvider struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	IDPMetadataURL string `json:"idp_metadata_url"`
	EntityID       string `json:"entity_id"`
	ACSURL         string `json:"acs_url"`
	AdminGroup     string `json:"admin_group,omitempty"`
	Enabled        bool   `json:"enabled"`
}

// ProviderStore handles OIDC and SAML provider persistence
type ProviderStore struct {
	db *DB
}

// NewProviderStore creates a new provider store
func NewProviderStore(db *DB) *ProviderStore {
	return &ProviderStore{db: db}
}

// OIDC Provider operations

func (s *ProviderStore) GetOIDCProviders(ctx context.Context) ([]*OIDCProvider, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, display_name, issuer, client_id, redirect_url, scopes, admin_group, is_enabled
		FROM oidc_providers
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*OIDCProvider
	for rows.Next() {
		var p OIDCProvider
		var scopesJSON []byte
		var adminGroup *string
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.Issuer, &p.ClientID, &p.RedirectURL, &scopesJSON, &adminGroup, &p.Enabled); err != nil {
			return nil, err
		}
		json.Unmarshal(scopesJSON, &p.Scopes)
		if adminGroup != nil {
			p.AdminGroup = *adminGroup
		}
		providers = append(providers, &p)
	}
	return providers, rows.Err()
}

func (s *ProviderStore) GetOIDCProvider(ctx context.Context, name string) (*OIDCProvider, error) {
	var p OIDCProvider
	var scopesJSON []byte
	var adminGroup *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, display_name, issuer, client_id, client_secret, redirect_url, scopes, admin_group, is_enabled
		FROM oidc_providers WHERE name = $1
	`, name).Scan(&p.ID, &p.Name, &p.DisplayName, &p.Issuer, &p.ClientID, &p.ClientSecret, &p.RedirectURL, &scopesJSON, &adminGroup, &p.Enabled)
	if err == pgx.ErrNoRows {
		return nil, ErrProviderNotFound
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(scopesJSON, &p.Scopes)
	if adminGroup != nil {
		p.AdminGroup = *adminGroup
	}
	return &p, nil
}

func (s *ProviderStore) CreateOIDCProvider(ctx context.Context, p *OIDCProvider) error {
	scopesJSON, _ := json.Marshal(p.Scopes)
	var adminGroup *string
	if p.AdminGroup != "" {
		adminGroup = &p.AdminGroup
	}
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO oidc_providers (name, display_name, issuer, client_id, client_secret, redirect_url, scopes, admin_group, is_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, p.Name, p.DisplayName, p.Issuer, p.ClientID, p.ClientSecret, p.RedirectURL, scopesJSON, adminGroup, p.Enabled)
	if err != nil && err.Error() == `ERROR: duplicate key value violates unique constraint "oidc_providers_name_key" (SQLSTATE 23505)` {
		return ErrProviderExists
	}
	return err
}

func (s *ProviderStore) UpdateOIDCProvider(ctx context.Context, name string, p *OIDCProvider) error {
	scopesJSON, _ := json.Marshal(p.Scopes)
	var adminGroup *string
	if p.AdminGroup != "" {
		adminGroup = &p.AdminGroup
	}

	var result pgx.Rows
	var err error

	if p.ClientSecret == "" {
		// Don't update the secret if not provided
		result, err = s.db.Pool.Query(ctx, `
			UPDATE oidc_providers
			SET display_name = $2, issuer = $3, client_id = $4, redirect_url = $5, scopes = $6, admin_group = $7, is_enabled = $8
			WHERE name = $1
			RETURNING id
		`, name, p.DisplayName, p.Issuer, p.ClientID, p.RedirectURL, scopesJSON, adminGroup, p.Enabled)
	} else {
		result, err = s.db.Pool.Query(ctx, `
			UPDATE oidc_providers
			SET display_name = $2, issuer = $3, client_id = $4, client_secret = $5, redirect_url = $6, scopes = $7, admin_group = $8, is_enabled = $9
			WHERE name = $1
			RETURNING id
		`, name, p.DisplayName, p.Issuer, p.ClientID, p.ClientSecret, p.RedirectURL, scopesJSON, adminGroup, p.Enabled)
	}
	if err != nil {
		return err
	}
	defer result.Close()

	if !result.Next() {
		return ErrProviderNotFound
	}
	return nil
}

func (s *ProviderStore) DeleteOIDCProvider(ctx context.Context, name string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM oidc_providers WHERE name = $1`, name)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProviderNotFound
	}
	return nil
}

// SAML Provider operations

func (s *ProviderStore) GetSAMLProviders(ctx context.Context) ([]*SAMLProvider, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, name, display_name, idp_metadata_url, entity_id, acs_url, admin_group, is_enabled
		FROM saml_providers
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*SAMLProvider
	for rows.Next() {
		var p SAMLProvider
		var adminGroup *string
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.IDPMetadataURL, &p.EntityID, &p.ACSURL, &adminGroup, &p.Enabled); err != nil {
			return nil, err
		}
		if adminGroup != nil {
			p.AdminGroup = *adminGroup
		}
		providers = append(providers, &p)
	}
	return providers, rows.Err()
}

func (s *ProviderStore) GetSAMLProvider(ctx context.Context, name string) (*SAMLProvider, error) {
	var p SAMLProvider
	var adminGroup *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, display_name, idp_metadata_url, entity_id, acs_url, admin_group, is_enabled
		FROM saml_providers WHERE name = $1
	`, name).Scan(&p.ID, &p.Name, &p.DisplayName, &p.IDPMetadataURL, &p.EntityID, &p.ACSURL, &adminGroup, &p.Enabled)
	if err == pgx.ErrNoRows {
		return nil, ErrProviderNotFound
	}
	if err != nil {
		return nil, err
	}
	if adminGroup != nil {
		p.AdminGroup = *adminGroup
	}
	return &p, nil
}

func (s *ProviderStore) CreateSAMLProvider(ctx context.Context, p *SAMLProvider) error {
	var adminGroup *string
	if p.AdminGroup != "" {
		adminGroup = &p.AdminGroup
	}
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO saml_providers (name, display_name, idp_metadata_url, entity_id, acs_url, admin_group, is_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, p.Name, p.DisplayName, p.IDPMetadataURL, p.EntityID, p.ACSURL, adminGroup, p.Enabled)
	if err != nil && err.Error() == `ERROR: duplicate key value violates unique constraint "saml_providers_name_key" (SQLSTATE 23505)` {
		return ErrProviderExists
	}
	return err
}

func (s *ProviderStore) UpdateSAMLProvider(ctx context.Context, name string, p *SAMLProvider) error {
	var adminGroup *string
	if p.AdminGroup != "" {
		adminGroup = &p.AdminGroup
	}
	result, err := s.db.Pool.Exec(ctx, `
		UPDATE saml_providers
		SET display_name = $2, idp_metadata_url = $3, entity_id = $4, acs_url = $5, admin_group = $6, is_enabled = $7
		WHERE name = $1
	`, name, p.DisplayName, p.IDPMetadataURL, p.EntityID, p.ACSURL, adminGroup, p.Enabled)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProviderNotFound
	}
	return nil
}

func (s *ProviderStore) DeleteSAMLProvider(ctx context.Context, name string) error {
	result, err := s.db.Pool.Exec(ctx, `DELETE FROM saml_providers WHERE name = $1`, name)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProviderNotFound
	}
	return nil
}
