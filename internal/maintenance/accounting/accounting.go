// Package accounting performs operational accounting maintenance.
package accounting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gobcn/radius-director/internal/generator"
)

const staleSessionTerminateCause = "Stale-Session"

// Execer is the database capability required by stale-session maintenance.
// *sql.DB satisfies this interface.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Result summarizes one accounting maintenance pass.
type Result struct {
	PoliciesProcessed int
	SessionsClosed    int64
}

// Runner performs a single accounting maintenance pass against a tenant's
// accounting database. Now is injectable so cutoff calculations are
// deterministic in tests.
type Runner struct {
	DB  Execer
	Now func() time.Time
}

// Run closes stale accounting sessions for each enabled NAS accounting policy.
// Policies with no stale-session timeout are ignored. Independent policy
// failures are collected so later policies can still be processed.
func (r Runner) Run(ctx context.Context, policies []generator.NASAccountingPolicy) (Result, error) {
	var result Result
	if r.DB == nil {
		return result, errors.New("accounting maintenance database is required")
	}

	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	maintenanceTime := now()

	var errs []error
	for _, policy := range policies {
		if policy.StaleSessionTimeout == nil {
			continue
		}

		result.PoliciesProcessed++
		cutoff := maintenanceTime.Add(-*policy.StaleSessionTimeout)
		databaseResult, err := r.DB.ExecContext(ctx, closeStaleSessionsQuery, staleSessionTerminateCause, policy.IPAddress, cutoff)
		if err != nil {
			errs = append(errs, fmt.Errorf("NAS device %q: %w", policy.NASDeviceIdentifier, err))
			continue
		}

		rowsAffected, err := databaseResult.RowsAffected()
		if err != nil {
			errs = append(errs, fmt.Errorf("NAS device %q: determine closed session count: %w", policy.NASDeviceIdentifier, err))
			continue
		}
		result.SessionsClosed += rowsAffected
	}

	return result, errors.Join(errs...)
}

// closeStaleSessionsQuery is intentionally MySQL/FreeRADIUS accounting-schema
// specific. Domain and generated models remain independent of radacct details.
// The stop time records the last accounting activity, not maintenance time.
const closeStaleSessionsQuery = `UPDATE radacct
SET acctstoptime = acctupdatetime,
    acctterminatecause = ?
WHERE nasipaddress = ?
  AND acctstoptime IS NULL
  AND acctupdatetime < ?`
