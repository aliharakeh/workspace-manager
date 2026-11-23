package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMoveWorkspaceLogosToFiles(t *testing.T) {
	t.Setenv("WM_DATA_DIR", t.TempDir())
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if err := applySchema(sqlDB); err != nil {
		t.Fatal(err)
	}
	png := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
	// Values an earlier build may have left in icon: an inline data URL, a
	// stray word, and a data URL that is not an image.
	if _, err := sqlDB.Exec(`INSERT INTO workspaces (name, icon) VALUES ('a', ?), ('b', 'rocket'), ('c', 'data:image/png;base64,aGk='), ('d', NULL)`, png); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // a second run leaves stored names alone
		if err := applySchema(sqlDB); err != nil {
			t.Fatalf("apply #%d: %v", i+1, err)
		}
	}
	icons := map[string]*string{}
	rows, err := sqlDB.Query(`SELECT name, icon FROM workspaces`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		var icon *string
		if err := rows.Scan(&name, &icon); err != nil {
			t.Fatal(err)
		}
		icons[name] = icon
	}
	_ = rows.Close()

	if icons["a"] == nil || !IsLogoName(*icons["a"]) {
		t.Fatalf("a data URL should become a logo file name: %v", icons["a"])
	}
	if _, err := os.Stat(filepath.Join(LogoDir(), *icons["a"])); err != nil {
		t.Errorf("logo file: %v", err)
	}
	for _, n := range []string{"b", "c", "d"} {
		if icons[n] != nil {
			t.Errorf("%s: an unusable icon should be cleared: %q", n, *icons[n])
		}
	}
	if entries, _ := os.ReadDir(LogoDir()); len(entries) != 1 {
		t.Errorf("only one logo file should be written: %d", len(entries))
	}
}
