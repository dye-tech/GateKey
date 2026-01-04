# Geo-Fencing

GateKey supports IP-based geo-fencing to restrict VPN access based on the client's source IP address. This uses a **whitelist approach** - only IPs/CIDRs explicitly allowed can connect; all others are blocked.

## Overview

Geo-fencing provides an additional layer of security by controlling where VPN connections can originate from. This is useful for:

- Restricting VPN access to specific countries or regions
- Limiting connections to known IP ranges (office networks, cloud providers)
- Compliance requirements that mandate geographic restrictions
- Zero-trust security models

## How It Works

### Whitelist Model

Geo-fencing uses a whitelist-only model:

1. Create rules that define allowed IP ranges (e.g., US IP blocks, office IPs)
2. Assign rules globally, to groups, or to individual users
3. When geo-fencing is enabled, only connections from allowed IPs are permitted
4. Connections from unlisted IPs are blocked (or logged in audit mode)

### Rule Hierarchy

Rules follow a hierarchical priority (most specific wins):

| Priority | Level | Description |
|----------|-------|-------------|
| 1 (highest) | User | Rules assigned directly to the user |
| 2 | Group | Rules assigned to any group the user belongs to |
| 3 (lowest) | Global | Default rules that apply to everyone |

If a user has user-specific rules, only those rules are evaluated. If not, group rules are checked. If neither exists, global rules apply.

**Important:** If geo-fencing is enabled but no rules apply to a user (no user/group/global rules), the connection is blocked.

## Configuration

### Enabling Geo-Fencing

1. Navigate to **Administration > Geo-Fencing**
2. Toggle **Enable Geo-Fencing**
3. Select the enforcement mode:
   - **Enforce**: Block connections that don't match allowed IPs
   - **Audit**: Log violations but allow connections (for testing)

### Creating Rules

1. Go to the **Rules** tab
2. Click **Create Rule**
3. Configure the rule:
   - **Name**: Descriptive name (e.g., "US IP Ranges")
   - **Description**: Optional explanation
   - **IP Range**: CIDR notation (e.g., `8.0.0.0/8`, `203.0.113.0/24`)
   - **Active**: Enable/disable the rule
4. Click **Save**

### Assigning Rules

#### Global Rules
Global rules apply to all users unless overridden by user or group rules.

1. Go to the **Global Rules** tab
2. Click **Add Rule**
3. Select the rule to add
4. The rule now applies to all users

#### Group Rules
Group rules apply to members of the specified group.

1. Go to the **Group Rules** tab
2. Search for and select a group
3. Click **Add Rule**
4. Select the rule to assign

#### User Rules
User rules override all other rules for the specific user.

1. Go to the **User Rules** tab
2. Search for and select a user
3. Click **Add Rule**
4. Select the rule to assign

## Enforcement Points

Geo-fencing is enforced at two points for defense-in-depth:

1. **Config Verification** (`/api/v1/gateway/verify`): When OpenVPN verifies a client certificate
2. **Connection Establishment** (`/api/v1/gateway/connect`): When a client connects

This ensures connections are blocked even if someone obtains a valid certificate.

## API Endpoints

### Settings

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/admin/geo-fence/settings` | Get geo-fence settings |
| PUT | `/api/v1/admin/geo-fence/settings` | Update settings (enabled, enforceMode) |

### Rules

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/admin/geo-fence/rules` | List all rules |
| POST | `/api/v1/admin/geo-fence/rules` | Create rule |
| GET | `/api/v1/admin/geo-fence/rules/:id` | Get rule details |
| PUT | `/api/v1/admin/geo-fence/rules/:id` | Update rule |
| DELETE | `/api/v1/admin/geo-fence/rules/:id` | Delete rule |

### Global Rules

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/admin/geo-fence/global` | List global assignments |
| POST | `/api/v1/admin/geo-fence/global` | Add global assignment |
| DELETE | `/api/v1/admin/geo-fence/global/:ruleId` | Remove global assignment |

### User Rules

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/admin/geo-fence/users/:userId/rules` | List user's rules |
| POST | `/api/v1/admin/geo-fence/users/:userId/rules` | Add user rule |
| DELETE | `/api/v1/admin/geo-fence/users/:userId/rules/:ruleId` | Remove user rule |

### Group Rules

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/admin/geo-fence/groups/:groupName/rules` | List group's rules |
| POST | `/api/v1/admin/geo-fence/groups/:groupName/rules` | Add group rule |
| DELETE | `/api/v1/admin/geo-fence/groups/:groupName/rules/:ruleId` | Remove group rule |

## Example Configurations

### Allow Only US IPs

1. Create rules for major US IP ranges:
   - "US Block 1" - `3.0.0.0/8`
   - "US Block 2" - `8.0.0.0/8`
   - Add more as needed for your users
2. Add all rules as global rules
3. Enable geo-fencing in enforce mode

### Allow Office + Home for Remote Workers

1. Create rule: "Office Network" - `203.0.113.0/24`
2. Create rule: "Alice Home" - `198.51.100.50/32`
3. Add "Office Network" as a global rule
4. Assign "Alice Home" to user Alice
5. Alice can connect from office or home; others only from office

### Allow Any IP for Specific User

1. Create rule: "Any IP" - `0.0.0.0/0`
2. Assign this rule to the specific user
3. This user bypasses geo-fencing (user rules override global)

### Restrict Contractors to Office Only

1. Create rule: "Office Only" - `203.0.113.0/24`
2. Assign to the "contractors" group
3. Contractors can only connect from the office network

## Best Practices

1. **Start in Audit Mode**: Test your rules in audit mode first to identify issues before enforcing
2. **Use Groups for Scale**: Assign rules to groups rather than individual users when possible
3. **Document IP Ranges**: Keep records of what IP ranges correspond to which locations/providers
4. **Monitor Blocked Connections**: Review logs for legitimate users being blocked
5. **Plan for Mobile Users**: Consider that mobile users may connect from various IPs
6. **Use Specific Rules**: Prefer specific CIDR ranges over broad ones to minimize false positives

## Troubleshooting

### User Can't Connect

1. Check if geo-fencing is enabled
2. Verify the user's current IP address
3. Check what rules apply to the user (user > group > global)
4. Verify at least one rule covers the user's IP
5. Check audit logs for blocked connection attempts

### Rule Not Taking Effect

1. Verify the rule is active (not disabled)
2. Verify the rule is assigned (global, group, or user)
3. Check rule priority - user rules override group/global
4. Regenerate VPN config after rule changes

### All Connections Blocked

1. Verify at least one global rule exists
2. Check the rule CIDR notation is correct
3. Verify rules are assigned and active
4. Try switching to audit mode to debug
