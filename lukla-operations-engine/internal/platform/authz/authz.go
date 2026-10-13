// Package authz provides RBAC permission evaluation and tenant-scope enforcement.
// Backend is the single source of truth for all authorization decisions.
// UI permission checks are only for experience; backend always re-validates.
package authz

import (
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Permission represents a granular action a role may perform.
type Permission string

const (
	PermViewInventory       Permission = "inventory:view"
	PermCreateIssue         Permission = "inventory:create_issue"
	PermVerifyOutput        Permission = "production:verify_output"
	PermInspectQuality      Permission = "quality:inspect"
	PermCalculatePayroll    Permission = "payroll:calculate"
	PermLockPayroll         Permission = "payroll:lock"
	PermManageRates         Permission = "ratecard:manage"
	PermManageRoles         Permission = "access:manage_roles"
	PermViewClientPortal    Permission = "portal:view"
)

// rolePermissions maps role names to their allowed permissions.
// TODO: load from database or config to allow runtime updates.
var rolePermissions = map[string][]Permission{
	"admin":             {PermViewInventory, PermCreateIssue, PermVerifyOutput, PermInspectQuality, PermCalculatePayroll, PermLockPayroll, PermManageRates, PermManageRoles},
	"supervisor":        {PermViewInventory, PermVerifyOutput},
	"qc_inspector":      {PermInspectQuality},
	"warehouse":         {PermViewInventory, PermCreateIssue},
	"payroll_officer":   {PermCalculatePayroll, PermLockPayroll},
	"client":            {PermViewClientPortal},
}

// HasPermission returns true if any of the user's roles include the required permission.
func HasPermission(claims authn.Claims, required Permission) bool {
	for _, role := range claims.Roles {
		for _, perm := range rolePermissions[role] {
			if perm == required {
				return true
			}
		}
	}
	return false
}

// HasTenantAccess returns true if the claims allow access to the given tenantID.
func HasTenantAccess(claims authn.Claims, tenantID string) bool {
	// Admins and internal roles may access all tenants; client users are scoped to their own.
	if claims.TenantID == "" {
		return false
	}
	return claims.TenantID == tenantID || claims.TenantID == "internal"
}
