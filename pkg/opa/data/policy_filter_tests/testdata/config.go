package testdata

import "embed"

//go:embed application-test.yml
var ConfigFS embed.FS

//go:embed bundles/model_a
var ModelABundleFS embed.FS

//go:embed bundles/model_b
var ModelBBundleFS embed.FS

//go:embed bundles/model_c
var ModelCBundleFS embed.FS

//go:embed bundles/model_d
var ModelDBundleFS embed.FS
