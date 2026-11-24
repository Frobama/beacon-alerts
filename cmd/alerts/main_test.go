package main

import "testing"

func TestVersionSaleCero(t *testing.T) {
	if code := dispatch([]string{"version"}); code != 0 {
		t.Fatalf("código = %d", code)
	}
}

func TestMigrateSinDatabaseURLFalla(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if code := dispatch([]string{"migrate"}); code != 1 {
		t.Fatalf("código = %d, quería 1", code)
	}
}
