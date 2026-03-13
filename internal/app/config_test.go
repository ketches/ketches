package app

import "testing"

func TestBuildDBSourceUsesExplicitSource(t *testing.T) {
	t.Setenv("DB_SOURCE", "custom-source")

	result := buildDBSource("postgres", "db", "5432", "ketches", "user", "pass", "disable")

	if result != "custom-source" {
		t.Fatalf("expected explicit DB_SOURCE to win, got %q", result)
	}
}

func TestBuildDBSourceBuildsPostgresDSN(t *testing.T) {
	result := buildDBSource("postgres", "db", "5432", "ketches", "user", "pass", "require")
	expected := "host=db port=5432 user=user password=pass dbname=ketches sslmode=require"

	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestBuildDBSourceBuildsMySQLDSN(t *testing.T) {
	result := buildDBSource("mysql", "db", "3306", "ketches", "user", "pass", "")
	expected := "user:pass@tcp(db:3306)/ketches?charset=utf8mb4&parseTime=True&loc=Local"

	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestBuildDBSourceUsesDBNameForSQLite(t *testing.T) {
	result := buildDBSource("sqlite", "", "", "custom.db", "", "", "")

	if result != "custom.db" {
		t.Fatalf("expected sqlite db name to be used, got %q", result)
	}
}
