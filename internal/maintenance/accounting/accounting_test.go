package accounting

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gobcn/radius-director/internal/generator"
)

type execCall struct {
	query string
	args  []any
}

type fakeExecer struct {
	calls       []execCall
	results     []sql.Result
	errors      []error
	resultIndex int
}

func (f *fakeExecer) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	f.calls = append(f.calls, execCall{query: query, args: args})
	index := f.resultIndex
	f.resultIndex++
	if index < len(f.errors) && f.errors[index] != nil {
		return nil, f.errors[index]
	}
	if index < len(f.results) {
		return f.results[index], nil
	}
	return fakeResult(0), nil
}

type fakeResult int64

func (f fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (f fakeResult) RowsAffected() (int64, error) { return int64(f), nil }

type rowsAffectedErrorResult struct{}

func (rowsAffectedErrorResult) LastInsertId() (int64, error) { return 0, nil }
func (rowsAffectedErrorResult) RowsAffected() (int64, error) {
	return 0, errors.New("rows affected unavailable")
}

func duration(value time.Duration) *time.Duration { return &value }

func TestRunnerClosesStaleSessionsUsingPerNASCutoff(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	db := &fakeExecer{results: []sql.Result{fakeResult(2), fakeResult(1)}}
	runner := Runner{DB: db, Now: func() time.Time { return now }}

	policies := []generator.NASAccountingPolicy{
		{NASDeviceIdentifier: "nas-a", IPAddress: "172.16.160.3", StaleSessionTimeout: duration(20 * time.Minute)},
		{NASDeviceIdentifier: "nas-disabled", IPAddress: "172.16.160.4"},
		{NASDeviceIdentifier: "nas-b", IPAddress: "172.16.160.50", StaleSessionTimeout: duration(time.Hour)},
	}

	result, err := runner.Run(context.Background(), policies)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.PoliciesProcessed != 2 || result.SessionsClosed != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(db.calls) != 2 {
		t.Fatalf("got %d database calls, want 2", len(db.calls))
	}

	wantFirstArgs := []any{staleSessionTerminateCause, "172.16.160.3", now.Add(-20 * time.Minute)}
	wantSecondArgs := []any{staleSessionTerminateCause, "172.16.160.50", now.Add(-time.Hour)}
	if !reflect.DeepEqual(db.calls[0].args, wantFirstArgs) {
		t.Fatalf("first args = %#v, want %#v", db.calls[0].args, wantFirstArgs)
	}
	if !reflect.DeepEqual(db.calls[1].args, wantSecondArgs) {
		t.Fatalf("second args = %#v, want %#v", db.calls[1].args, wantSecondArgs)
	}
}

func TestRunnerUsesAtomicIdempotentUpdate(t *testing.T) {
	db := &fakeExecer{}
	runner := Runner{DB: db, Now: func() time.Time { return time.Unix(0, 0) }}
	_, err := runner.Run(context.Background(), []generator.NASAccountingPolicy{
		{NASDeviceIdentifier: "nas-a", IPAddress: "192.0.2.1", StaleSessionTimeout: duration(time.Minute)},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	query := db.calls[0].query
	for _, required := range []string{
		"acctstoptime = acctupdatetime",
		"acctterminatecause = ?",
		"nasipaddress = ?",
		"acctstoptime IS NULL",
		"acctupdatetime < ?",
	} {
		if !strings.Contains(query, required) {
			t.Fatalf("query does not contain %q:\n%s", required, query)
		}
	}
	if strings.Contains(strings.ToUpper(query), "NOW()") {
		t.Fatalf("query must use the Go-calculated cutoff, not database NOW():\n%s", query)
	}
}

func TestRunnerContinuesAfterIndependentPolicyFailure(t *testing.T) {
	db := &fakeExecer{
		errors:  []error{errors.New("first failed"), nil},
		results: []sql.Result{nil, fakeResult(4)},
	}
	runner := Runner{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }}

	result, err := runner.Run(context.Background(), []generator.NASAccountingPolicy{
		{NASDeviceIdentifier: "nas-a", IPAddress: "192.0.2.1", StaleSessionTimeout: duration(time.Minute)},
		{NASDeviceIdentifier: "nas-b", IPAddress: "192.0.2.2", StaleSessionTimeout: duration(time.Minute)},
	})
	if err == nil || !strings.Contains(err.Error(), `NAS device "nas-a"`) {
		t.Fatalf("error = %v, want contextual first-policy error", err)
	}
	if len(db.calls) != 2 {
		t.Fatalf("got %d calls, want maintenance to continue to second policy", len(db.calls))
	}
	if result.SessionsClosed != 4 {
		t.Fatalf("SessionsClosed = %d, want 4", result.SessionsClosed)
	}
}

func TestRunnerReportsRowsAffectedFailureAndContinues(t *testing.T) {
	db := &fakeExecer{results: []sql.Result{rowsAffectedErrorResult{}, fakeResult(1)}}
	runner := Runner{DB: db, Now: func() time.Time { return time.Unix(1000, 0) }}

	result, err := runner.Run(context.Background(), []generator.NASAccountingPolicy{
		{NASDeviceIdentifier: "nas-a", IPAddress: "192.0.2.1", StaleSessionTimeout: duration(time.Minute)},
		{NASDeviceIdentifier: "nas-b", IPAddress: "192.0.2.2", StaleSessionTimeout: duration(time.Minute)},
	})
	if err == nil || !strings.Contains(err.Error(), "determine closed session count") {
		t.Fatalf("error = %v, want rows-affected error", err)
	}
	if result.SessionsClosed != 1 {
		t.Fatalf("SessionsClosed = %d, want 1", result.SessionsClosed)
	}
}

func TestRunnerRequiresDatabase(t *testing.T) {
	_, err := (Runner{}).Run(context.Background(), nil)
	if err == nil {
		t.Fatal("Run returned nil error without database")
	}
}
