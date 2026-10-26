package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func sendLeadToGoogleAppsScript(cfg Config, lead Lead) error {
	lead.ScriptToken = cfg.GoogleAppsScriptToken
	lead.ScriptToken = cfg.GoogleAppsScriptToken

	if cfg.GoogleAppsScriptURL == "" {
		return fmt.Errorf("GOOGLE_APPS_SCRIPT_URL is empty")
	}

	if !strings.Contains(cfg.GoogleAppsScriptURL, "script.google.com/macros/s/") {
		return fmt.Errorf("GOOGLE_APPS_SCRIPT_URL is not an Apps Script Web App URL")
	}

	if !strings.HasSuffix(cfg.GoogleAppsScriptURL, "/exec") {
		return fmt.Errorf("GOOGLE_APPS_SCRIPT_URL must end with /exec")
	}

	jsonBody, err := json.Marshal(lead)
	if err != nil {
		return fmt.Errorf("marshal lead: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, cfg.GoogleAppsScriptURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create apps script request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send apps script request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("apps script returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed AppsScriptResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("decode apps script response: %w, raw: %s", err, string(body))
	}

	if !parsed.OK {
		return fmt.Errorf("apps script error: %s", parsed.Error)
	}

	log.Printf("Apps Script saved lead to sheet: %s", parsed.Sheet)
	return nil
}
