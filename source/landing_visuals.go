package main

import (
	"fmt"
	"strconv"
	"strings"
)

func buildHeroStyleVars(settings SiteSettings) string {
	if settings.HeroOverlayOpacity == "" {
		return "--hero-scene-overlay-opacity:0;"
	}
	preset := normalizedChoice(settings.HeroScenePreset, []string{"premium-dark", "balanced", "contrast", "soft"}, "premium-dark")
	defaults := map[string]map[string]string{
		"premium-dark": {
			"brightness": "0.74",
			"contrast":   "1.12",
			"saturation": "0.92",
			"blur":       "0",
			"opacity":    "0.94",
			"scale":      "1.04",
			"overlay":    "0.78",
		},
		"balanced": {
			"brightness": "0.84",
			"contrast":   "1.05",
			"saturation": "1.00",
			"blur":       "0",
			"opacity":    "0.98",
			"scale":      "1.02",
			"overlay":    "0.72",
		},
		"contrast": {
			"brightness": "0.78",
			"contrast":   "1.18",
			"saturation": "0.98",
			"blur":       "0",
			"opacity":    "0.96",
			"scale":      "1.05",
			"overlay":    "0.76",
		},
		"soft": {
			"brightness": "0.88",
			"contrast":   "1.00",
			"saturation": "0.94",
			"blur":       "0.6",
			"opacity":    "0.94",
			"scale":      "1.02",
			"overlay":    "0.68",
		},
	}

	overlay := clampedNumberString(settings.HeroOverlayOpacity, defaults[preset]["overlay"], 0, 1, 2)
	return fmt.Sprintf("--hero-scene-overlay-opacity:%s;", overlay)
}

func buildHeroSceneImageStyle(settings SiteSettings) string {
	if settings.HeroScenePreset == "" && settings.HeroSceneBrightness == "" && settings.HeroSceneContrast == "" && settings.HeroSceneSaturation == "" && settings.HeroSceneBlur == "" && settings.HeroSceneOpacity == "" && settings.HeroSceneScale == "" {
		return "filter:none;"
	}
	preset := normalizedChoice(settings.HeroScenePreset, []string{"premium-dark", "balanced", "contrast", "soft"}, "premium-dark")
	defaults := map[string]map[string]string{
		"premium-dark": {"brightness": "0.74", "contrast": "1.12", "saturation": "0.92", "blur": "0", "opacity": "0.94", "scale": "1.04"},
		"balanced":     {"brightness": "0.84", "contrast": "1.05", "saturation": "1.00", "blur": "0", "opacity": "0.98", "scale": "1.02"},
		"contrast":     {"brightness": "0.78", "contrast": "1.18", "saturation": "0.98", "blur": "0", "opacity": "0.96", "scale": "1.05"},
		"soft":         {"brightness": "0.88", "contrast": "1.00", "saturation": "0.94", "blur": "0.6", "opacity": "0.94", "scale": "1.02"},
	}
	brightness := clampedNumberString(settings.HeroSceneBrightness, defaults[preset]["brightness"], 0.4, 1.6, 2)
	contrast := clampedNumberString(settings.HeroSceneContrast, defaults[preset]["contrast"], 0.4, 1.8, 2)
	saturation := clampedNumberString(settings.HeroSceneSaturation, defaults[preset]["saturation"], 0, 2, 2)
	blur := clampedNumberString(settings.HeroSceneBlur, defaults[preset]["blur"], 0, 8, 1)
	opacity := clampedNumberString(settings.HeroSceneOpacity, defaults[preset]["opacity"], 0.2, 1, 2)
	scale := clampedNumberString(settings.HeroSceneScale, defaults[preset]["scale"], 1, 1.2, 2)
	return fmt.Sprintf("filter:brightness(%s) contrast(%s) saturate(%s) blur(%spx); opacity:%s; transform:scale(%s); transform-origin:center center;", brightness, contrast, saturation, blur, opacity, scale)
}

func buildHeroPersonLayerStyle(settings SiteSettings) string {
	preset := normalizedChoice(settings.HeroPersonPreset, []string{"natural", "crisp", "soft"}, "natural")
	defaults := map[string]map[string]string{
		"natural": {"x": "-18", "y": "0"},
		"crisp":   {"x": "-14", "y": "0"},
		"soft":    {"x": "-16", "y": "4"},
	}
	x := clampedNumberString(settings.HeroPersonPositionX, defaults[preset]["x"], -40, 20, 0)
	y := clampedNumberString(settings.HeroPersonPositionY, defaults[preset]["y"], -80, 80, 0)
	return fmt.Sprintf("transform:translateX(%s%%); bottom:%spx;", x, y)
}

func buildHeroPersonImageStyle(settings SiteSettings) string {
	preset := normalizedChoice(settings.HeroPersonPreset, []string{"natural", "crisp", "soft"}, "natural")
	defaults := map[string]map[string]string{
		"natural": {"brightness": "1.00", "contrast": "1.00", "saturation": "1.00", "opacity": "1.00", "scale": "1.00", "shadow": "0 28px 52px rgba(2, 8, 23, 0.34)"},
		"crisp":   {"brightness": "1.02", "contrast": "1.08", "saturation": "1.02", "opacity": "1.00", "scale": "1.03", "shadow": "0 36px 72px rgba(2, 8, 23, 0.42)"},
		"soft":    {"brightness": "0.98", "contrast": "0.98", "saturation": "0.96", "opacity": "0.94", "scale": "1.00", "shadow": "0 24px 44px rgba(2, 8, 23, 0.28)"},
	}
	brightness := clampedNumberString(settings.HeroPersonBrightness, defaults[preset]["brightness"], 0.5, 1.6, 2)
	contrast := clampedNumberString(settings.HeroPersonContrast, defaults[preset]["contrast"], 0.5, 1.8, 2)
	saturation := clampedNumberString(settings.HeroPersonSaturation, defaults[preset]["saturation"], 0, 2, 2)
	opacity := clampedNumberString(settings.HeroPersonOpacity, defaults[preset]["opacity"], 0.2, 1, 2)
	scale := clampedNumberString(settings.HeroPersonScale, defaults[preset]["scale"], 0.8, 1.2, 2)
	shadow := sanitizeShadow(settings.HeroPersonShadow)
	if shadow == "" {
		shadow = defaults[preset]["shadow"]
	}
	return fmt.Sprintf("filter:brightness(%s) contrast(%s) saturate(%s) drop-shadow(%s); opacity:%s; transform:scale(%s); transform-origin:bottom center;", brightness, contrast, saturation, shadow, opacity, scale)
}

func buildAreasStyleVars(settings SiteSettings) string {
	preset := normalizedChoice(settings.AreasMapPreset, []string{"clean", "contrast", "muted"}, "clean")
	defaults := map[string]string{"clean": "0.72", "contrast": "0.78", "muted": "0.66"}
	overlay := clampedNumberString(settings.AreasOverlayOpacity, defaults[preset], 0, 1, 2)
	return fmt.Sprintf("--areas-overlay-opacity:%s;", overlay)
}

func buildAreasMapImageStyle(settings SiteSettings) string {
	preset := normalizedChoice(settings.AreasMapPreset, []string{"clean", "contrast", "muted"}, "clean")
	defaults := map[string]map[string]string{
		"clean":    {"brightness": "1.02", "contrast": "1.04", "saturation": "1.02", "blur": "0", "opacity": "0.94", "scale": "1.00"},
		"contrast": {"brightness": "0.98", "contrast": "1.12", "saturation": "1.08", "blur": "0", "opacity": "0.94", "scale": "1.02"},
		"muted":    {"brightness": "1.00", "contrast": "1.00", "saturation": "0.78", "blur": "0", "opacity": "0.90", "scale": "1.00"},
	}
	brightness := clampedNumberString(settings.AreasMapBrightness, defaults[preset]["brightness"], 0.4, 1.6, 2)
	contrast := clampedNumberString(settings.AreasMapContrast, defaults[preset]["contrast"], 0.4, 1.8, 2)
	saturation := clampedNumberString(settings.AreasMapSaturation, defaults[preset]["saturation"], 0, 2, 2)
	blur := clampedNumberString(settings.AreasMapBlur, defaults[preset]["blur"], 0, 8, 1)
	opacity := clampedNumberString(settings.AreasMapOpacity, defaults[preset]["opacity"], 0.2, 1, 2)
	scale := clampedNumberString(settings.AreasMapScale, defaults[preset]["scale"], 0.9, 1.15, 2)
	return fmt.Sprintf("filter:brightness(%s) contrast(%s) saturate(%s) blur(%spx); opacity:%s; transform:scale(%s); transform-origin:center center;", brightness, contrast, saturation, blur, opacity, scale)
}

func buildServiceImageStyle(settings SiteSettings) string {
	preset := normalizedChoice(settings.ServiceImagePreset, []string{"clean", "crisp", "cinematic"}, "clean")
	defaults := map[string]map[string]string{
		"clean":     {"brightness": "1.00", "contrast": "1.03", "saturation": "1.00", "blur": "0", "opacity": "1.00", "scale": "1.00"},
		"crisp":     {"brightness": "1.02", "contrast": "1.08", "saturation": "1.05", "blur": "0", "opacity": "1.00", "scale": "1.02"},
		"cinematic": {"brightness": "0.96", "contrast": "1.10", "saturation": "0.96", "blur": "0", "opacity": "0.98", "scale": "1.03"},
	}
	brightness := clampedNumberString(settings.ServiceImageBrightness, defaults[preset]["brightness"], 0.4, 1.6, 2)
	contrast := clampedNumberString(settings.ServiceImageContrast, defaults[preset]["contrast"], 0.4, 1.8, 2)
	saturation := clampedNumberString(settings.ServiceImageSaturation, defaults[preset]["saturation"], 0, 2, 2)
	blur := clampedNumberString(settings.ServiceImageBlur, defaults[preset]["blur"], 0, 8, 1)
	opacity := clampedNumberString(settings.ServiceImageOpacity, defaults[preset]["opacity"], 0.2, 1, 2)
	scale := clampedNumberString(settings.ServiceImageScale, defaults[preset]["scale"], 1, 1.12, 2)
	return fmt.Sprintf("filter:brightness(%s) contrast(%s) saturate(%s) blur(%spx); opacity:%s; transform:scale(%s); transform-origin:center center;", brightness, contrast, saturation, blur, opacity, scale)
}

func normalizedChoice(raw string, allowed []string, fallback string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	for _, item := range allowed {
		if raw == item {
			return raw
		}
	}
	return fallback
}

func clampedNumberString(raw, fallback string, min, max float64, precision int) string {
	value := fallback
	if strings.TrimSpace(raw) != "" {
		value = strings.TrimSpace(raw)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		parsed, _ = strconv.ParseFloat(fallback, 64)
	}
	if parsed < min {
		parsed = min
	}
	if parsed > max {
		parsed = max
	}
	format := fmt.Sprintf("%%.%df", precision)
	formatted := fmt.Sprintf(format, parsed)
	if precision > 0 {
		formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	}
	return formatted
}

func sanitizeShadow(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	replacer := strings.NewReplacer(";", "", "\"", "", "'", "", "<", "", ">", "")
	return replacer.Replace(raw)
}
