package database

import (
	"os"
	"path/filepath"
	"testing"
)

// Test that Init returns same instance on multiple calls
func TestInitIdempotent(t *testing.T) {
	ResetForTest()
	tempDir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	db1, err := Init("test1.db")
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	db2, err := Init("test1.db")
	if err != nil {
		t.Fatalf("second init failed: %v", err)
	}
	if db1 != db2 {
		t.Error("expected same instance on repeated Init calls")
	}
}

// Test that providing models triggers AutoMigrate and creates file
func TestInitWithModelsCreatesFile(t *testing.T) {
	ResetForTest()
	tempDir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	type Dummy struct{ ID int }
	_, err := Init("models.db", &Dummy{})
	if err != nil {
		t.Fatalf("init with model failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(".runtime", "models.db")); err != nil {
		if os.IsNotExist(err) {
			t.Errorf("expected db file created: %v", err)
		} else {
			t.Fatalf("stat error: %v", err)
		}
	}
}

// Test Get before Init returns nil
func TestGetBeforeInitReturnsNil(t *testing.T) {
	ResetForTest()
	if Get() != nil {
		t.Error("expected nil before Init")
	}
}
