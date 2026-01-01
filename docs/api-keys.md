# API Keys

API keys provide a way to authenticate with GateKey without using browser-based SSO login. They are ideal for:

- Automated scripts and CI/CD pipelines
- Headless servers without browser access
- Service accounts and integrations
- Admin CLI automation

## API Key Format

API keys use the following format:

```
gk_<base64-encoded-random-bytes>
```

Example: `gk_dGhpcyBpcyBhIHNhbXBsZSBhcGkga2V5IGZvciBkZW1v...`

**Security Notes:**
- The full API key is only shown **once** at creation time
- Store API keys securely (password managers, secrets management systems)
- API keys are stored as SHA-256 hashes in the database
- Only the key prefix (first 12 characters) is shown for identification

## Creating API Keys

### Via Web UI

1. Navigate to your **User Profile** (click your name in the top right)
2. Go to the **API Keys** tab
3. Click **Create API Key**
4. Enter a descriptive name (e.g., "CI/CD Pipeline", "Laptop CLI")
5. Optionally set an expiration time
6. Click **Create**
7. **Copy and save the API key immediately** - it won't be shown again

### Via Admin CLI

Administrators can create API keys for any user:

```bash
# Create API key for yourself
gatekey-admin api-key create "My CLI Key"

# Create API key for another user
gatekey-admin api-key create "Service Account Key" --user user@example.com

# Create API key with expiration
gatekey-admin api-key create "Temp Key" --expires 30d

# Create API key with limited scopes
gatekey-admin api-key create "Read Only" --scopes read:gateways,read:networks
```

## Using API Keys

### GateKey Client (gatekey)

```bash
# Login with API key
gatekey login --api-key gk_your_api_key_here

# The API key is stored in your config
# Subsequent commands use it automatically
gatekey connect
gatekey list
```

### Admin CLI (gatekey-admin)

```bash
# Login with API key
gatekey-admin login --api-key gk_your_api_key_here

# Or pass it per-command
gatekey-admin --api-key gk_your_api_key_here gateway list
```

### Direct API Access

```bash
# Use in Authorization header
curl -H "Authorization: Bearer gk_your_api_key_here" \
  https://vpn.example.com/api/v1/gateways
```

## Managing API Keys

### Listing API Keys

**Web UI:**
- Navigate to User Profile → API Keys
- See all your personal API keys

**Admin CLI:**
```bash
# List all API keys (admin)
gatekey-admin api-key list

# List API keys for a specific user
gatekey-admin api-key list --user user@example.com
```

### Revoking API Keys

**Web UI:**
1. Go to User Profile → API Keys
2. Click **Revoke** on the key you want to disable
3. Confirm the action

**Admin CLI:**
```bash
# Revoke a specific API key by ID
gatekey-admin api-key revoke <key-id>

# Revoke all keys for a user
gatekey-admin api-key revoke-all --user user@example.com
```

**Important:** Revoking an API key takes effect immediately. Any active sessions using that key will fail on the next request.

## API Key Properties

| Property | Description |
|----------|-------------|
| **ID** | Unique identifier (UUID) |
| **Name** | User-provided name for identification |
| **Description** | Optional description |
| **Key Prefix** | First 12 characters for identification (gk_xxxx...) |
| **Scopes** | Permissions granted to this key |
| **Created At** | When the key was created |
| **Expires At** | When the key will expire (optional) |
| **Last Used At** | Last time the key was used |
| **Last Used IP** | IP address of last use |
| **Is Revoked** | Whether the key has been revoked |
| **Admin Provisioned** | Whether an admin created this key |
| **Provisioned By** | Admin who created the key (if applicable) |

## Scopes

API keys can be limited to specific scopes:

| Scope | Description |
|-------|-------------|
| `*` | Full access (default for user-created keys) |
| `read:gateways` | List and view gateways |
| `write:gateways` | Create, update, delete gateways |
| `read:networks` | List and view networks |
| `write:networks` | Create, update, delete networks |
| `read:users` | List and view users |
| `write:users` | Manage users |
| `read:access-rules` | List and view access rules |
| `write:access-rules` | Manage access rules |
| `vpn:connect` | Generate VPN configurations |

## Expiration

API keys can be set to expire after a specific duration:

- `30d` - 30 days
- `90d` - 90 days
- `1y` - 1 year
- `never` - Never expires (not recommended for most use cases)

**Best Practice:** Set appropriate expiration times based on use case:
- Personal CLI usage: 90 days to 1 year
- CI/CD pipelines: 30-90 days with rotation
- Temporary access: 1-7 days

## Security Best Practices

1. **Principle of Least Privilege**
   - Use scopes to limit what each key can do
   - Create separate keys for different purposes

2. **Regular Rotation**
   - Rotate keys periodically (every 90 days recommended)
   - Immediately revoke compromised keys

3. **Secure Storage**
   - Never commit API keys to version control
   - Use secrets management (Vault, AWS Secrets Manager, etc.)
   - Store in environment variables, not command history

4. **Monitoring**
   - Review audit logs for unusual API key activity
   - Monitor "Last Used" timestamps for inactive keys
   - Revoke unused keys promptly

5. **Naming Conventions**
   - Use descriptive names that identify the key's purpose
   - Include the system/service name in the key name

## Audit Trail

All API key operations are logged:

- Key creation (who, when, what scopes)
- Key usage (which endpoints, from which IP)
- Key revocation (who revoked, reason)

View audit logs:
```bash
gatekey-admin audit list --filter api_key
```

## Troubleshooting

### "Invalid API key"
- Verify the key was copied correctly (no extra spaces)
- Check if the key has been revoked
- Verify the key hasn't expired

### "API key has been revoked"
- Create a new API key
- Contact an administrator if unexpected

### "API key has expired"
- Create a new API key with appropriate expiration
- Consider longer expiration for automation keys

### "Insufficient permissions"
- Check the key's scopes
- Request a key with broader scopes if needed
- Contact an administrator for scope changes

## See Also

- [Admin CLI Guide](admin-cli.md) - Full admin CLI documentation
- [Client Guide](client.md) - GateKey client documentation
- [API Reference](api.md) - REST API documentation
- [Security](security.md) - Security best practices
