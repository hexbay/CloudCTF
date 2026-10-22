package bootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestDialectorFromURI(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		driverName string
	}{
		{name: "default sqlite", uri: "", driverName: "sqlite"},
		{name: "sqlite", uri: "sqlite://:memory:", driverName: "sqlite"},
		{name: "sqlite alias", uri: "sqlite3://:memory:", driverName: "sqlite"},
		{name: "mysql", uri: "mysql://cloudctf:secret@tcp(localhost:3306)/cloudctf", driverName: "mysql"},
		{name: "postgres", uri: "postgres://cloudctf:secret@localhost:5432/cloudctf?sslmode=disable", driverName: "postgres"},
		{name: "postgresql alias", uri: "postgresql://cloudctf:secret@localhost:5432/cloudctf?sslmode=disable", driverName: "postgres"},
		{name: "trimmed uppercase scheme", uri: "  POSTGRES://cloudctf:secret@localhost:5432/cloudctf?sslmode=disable  ", driverName: "postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialector, err := dialectorFromURI(tt.uri)
			if err != nil {
				t.Fatalf("dialectorFromURI() error = %v", err)
			}
			if got := dialector.Name(); got != tt.driverName {
				t.Fatalf("dialector.Name() = %q, want %q", got, tt.driverName)
			}
		})
	}
}

func TestDialectorFromURIRejectsUnsupportedScheme(t *testing.T) {
	_, err := dialectorFromURI("sqlserver://localhost/cloudctf")
	if err == nil {
		t.Fatal("dialectorFromURI() expected an error")
	}
	if !strings.Contains(err.Error(), "postgres://") {
		t.Fatalf("error should list supported schemes: %v", err)
	}
}

func TestExternalDatabaseSchema(t *testing.T) {
	databaseURI := os.Getenv("CLOUDCTF_TEST_DATABASE_URI")
	if databaseURI == "" {
		t.Skip("CLOUDCTF_TEST_DATABASE_URI is not configured")
	}

	db, err := OpenDB(databaseURI)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	defer sqlDB.Close()

	if err := PrepareSchema(db); err != nil {
		t.Fatalf("PrepareSchema() error = %v", err)
	}
}
