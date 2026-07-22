package web

import "net/http"

func StartWebServer(configPath string, port int, logger *AppLogger) error {
	return startWebServer(configPath, port, logger)
}

func SaveWebConfig(configPath string, request webConfigUpdateRequest, logger *AppLogger) error {
	return saveWebConfig(configPath, request, logger)
}

func LoadWebRuntimeConfig(path string, logger *AppLogger) (Config, error) {
	return loadWebRuntimeConfig(path, logger)
}

func LoadWebDisplayConfig(path string, logger *AppLogger) (Config, error) {
	return loadWebDisplayConfig(path, logger)
}

func ValidateWebPlexConnectionRequest(request webConfigUpdateRequest) error {
	return validateWebPlexConnectionRequest(request)
}

func RunPosterGeneration(client *http.Client, cfg Config, selectedSections []plexSection, uploadPosters bool, labelTypeCollectionItems bool, missingPostersOnly bool, logger *AppLogger) error {
	return runPosterGeneration(client, cfg, selectedSections, uploadPosters, labelTypeCollectionItems, missingPostersOnly, logger)
}
