package db

// BoolInt converts a boolean to SQLite's integer representation.
func BoolInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
