package lib

import "regexp"

// BlueprintVarRe matches the placeholders a blueprint command may use.
var BlueprintVarRe = regexp.MustCompile(`\{\{\s*(app_name|folder_name|app_dir)\s*\}\}`)
