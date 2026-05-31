package config

import "fmt"

func publicViewerRateLimitConfigFromEnv() (PublicViewerRateLimitConfig, error) {
	enabled, err := boolFromEnv("SAFE_PUBLIC_VIEWER_RATE_LIMIT_ENABLED", defaultPublicViewerRateLimitEnabled)
	if err != nil {
		return PublicViewerRateLimitConfig{}, err
	}
	window, err := durationFromEnv("SAFE_PUBLIC_VIEWER_RATE_LIMIT_WINDOW", defaultPublicViewerRateLimitWindow)
	if err != nil {
		return PublicViewerRateLimitConfig{}, err
	}
	if enabled && window <= 0 {
		return PublicViewerRateLimitConfig{}, fmt.Errorf("parse SAFE_PUBLIC_VIEWER_RATE_LIMIT_WINDOW: duration must be positive when rate limiting is enabled")
	}

	pageLimit, err := nonNegativeIntFromEnv("SAFE_PUBLIC_VIEWER_RATE_LIMIT_PAGE", defaultPublicViewerRateLimitPageLimit)
	if err != nil {
		return PublicViewerRateLimitConfig{}, err
	}
	dataLimit, err := nonNegativeIntFromEnv("SAFE_PUBLIC_VIEWER_RATE_LIMIT_DATA", defaultPublicViewerRateLimitDataLimit)
	if err != nil {
		return PublicViewerRateLimitConfig{}, err
	}
	downloadLimit, err := nonNegativeIntFromEnv("SAFE_PUBLIC_VIEWER_RATE_LIMIT_DOWNLOAD", defaultPublicViewerRateLimitDownloadLimit)
	if err != nil {
		return PublicViewerRateLimitConfig{}, err
	}
	staticLimit, err := nonNegativeIntFromEnv("SAFE_PUBLIC_VIEWER_RATE_LIMIT_STATIC", defaultPublicViewerRateLimitStaticLimit)
	if err != nil {
		return PublicViewerRateLimitConfig{}, err
	}

	return PublicViewerRateLimitConfig{
		Enabled:       enabled,
		Window:        window,
		PageLimit:     pageLimit,
		DataLimit:     dataLimit,
		DownloadLimit: downloadLimit,
		StaticLimit:   staticLimit,
	}, nil
}
