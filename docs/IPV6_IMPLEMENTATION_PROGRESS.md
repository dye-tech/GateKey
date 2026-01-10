# IPv6 Implementation Progress

## Overview
Adding IPv6 compatibility across the GateKey platform while maintaining full backward compatibility with IPv4.

## Status: IN PROGRESS

## Implementation Phases

### Phase 1: Database Layer
- [ ] Update Go models with IPv6 fields
- [ ] Update database access functions for networks (cidr_v6)
- [ ] Update database access functions for access rules (value_v6)
- [ ] Update database access functions for geo-fence rules (ipv6_range)
- [ ] Update gateway models for IPv6 public IP
- [ ] Update mesh hub/spoke models for IPv6 subnets

### Phase 2: API Layer
- [ ] Update network handlers for IPv6 CIDR
- [ ] Update access rule handlers for IPv6 values
- [ ] Update geo-fence handlers for IPv6 ranges
- [ ] Update gateway handlers for IPv6 public IP
- [ ] Update mesh handlers for IPv6 subnets
- [ ] Update connection handlers for IPv6 VPN addresses

### Phase 3: Frontend UI
- [ ] Update API client TypeScript interfaces
- [ ] Add IPv6 fields to AdminNetworks page
- [ ] Add IPv6 fields to AdminAccessRules page
- [ ] Add IPv6 fields to AdminGeoFencing page
- [ ] Add IPv6 fields to AdminGateways page
- [ ] Add IPv6 fields to AdminMesh page
- [ ] Add IPv6 validation utilities

### Phase 4: Config Generation
- [ ] Update OpenVPN config generation for IPv6 routes
- [ ] Update WireGuard config generation for IPv6
- [ ] Update firewall rules for IPv6

### Phase 5: Testing
- [ ] Unit tests for IPv6 validation
- [ ] Unit tests for database layer
- [ ] Unit tests for API handlers
- [ ] Integration tests

### Phase 6: Deployment
- [ ] Build and push images
- [ ] Test in gatekey namespace
- [ ] Verify backward compatibility

## Files Modified

### Database Layer
| File | Status | Changes |
|------|--------|---------|
| internal/models/models.go | Pending | Add IPv6 fields to structs |
| internal/db/networks.go | Pending | Query cidr_v6 column |
| internal/db/access_rules.go | Pending | Query value_v6 column |
| internal/db/geo_fence.go | Pending | Query ipv6_range column |
| internal/db/gateways.go | Pending | Add IPv6 public IP support |
| internal/db/mesh.go | Pending | Add IPv6 subnet support |

### API Layer
| File | Status | Changes |
|------|--------|---------|
| internal/api/routes.go | Pending | Update handlers for IPv6 |
| internal/api/mesh_handlers.go | Pending | Update mesh handlers |
| internal/api/wireguard_handlers.go | Pending | Update WireGuard handlers |

### Frontend
| File | Status | Changes |
|------|--------|---------|
| web/src/api/client.ts | Pending | Add IPv6 interface fields |
| web/src/pages/AdminNetworks.tsx | Pending | IPv6 CIDR input |
| web/src/pages/AdminAccessRules.tsx | Pending | IPv6 value input |
| web/src/pages/AdminGeoFencing.tsx | Pending | IPv6 range input |
| web/src/pages/AdminGateways.tsx | Pending | IPv6 public IP input |
| web/src/pages/AdminMesh.tsx | Pending | IPv6 subnet inputs |

### Config Generation
| File | Status | Changes |
|------|--------|---------|
| internal/openvpn/config.go | Pending | IPv6 route generation |
| internal/wireguard/peer_manager.go | Pending | IPv6 allowed IPs |

## Database Schema (Already Exists - Migration 000042)
```sql
-- networks table
cidr_v6 TEXT  -- IPv6 CIDR notation

-- access_rules table
value_v6 TEXT  -- IPv6 IP/CIDR

-- geo_fence_rules table
ipv6_range TEXT  -- IPv6 ranges

-- connections table (from migration 000001)
vpn_ipv6 INET  -- Assigned VPN IPv6 address
```

## Key Design Decisions

1. **Dual-Stack Approach**: Support both IPv4 and IPv6 simultaneously
2. **Optional IPv6**: IPv6 fields are nullable to maintain backward compatibility
3. **Separate Fields**: Use separate database columns (e.g., cidr + cidr_v6) rather than combined
4. **Validation**: Use Go's net package which is already IPv6-compatible
5. **UI Collapsible**: IPv6 fields in UI should be in collapsible sections to avoid clutter

## Testing Strategy

1. **Unit Tests**: Test all IPv6 validation and parsing functions
2. **Database Tests**: Ensure IPv6 values are stored/retrieved correctly
3. **API Tests**: Test all endpoints with IPv6 values
4. **Integration Tests**: End-to-end tests with IPv6 configurations
5. **Backward Compatibility**: Ensure IPv4-only configs still work

## Notes

- Go's `net` package (ParseIP, ParseCIDR, IPNet) already supports IPv6
- PostgreSQL INET and CIDR types support IPv6
- WireGuard AllowedIPs already handles IPv6
- OpenVPN needs `route-ipv6` directives for IPv6 routes

---
Last Updated: 2026-01-09
