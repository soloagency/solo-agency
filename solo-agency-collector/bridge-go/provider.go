package main

// provider.go — unified Go port of tools/provider_openapi.py (content side)
// and outreach/tools/provider_openapi.py (218-line divergence). One adapter
// serves both roles: the operation-candidate map and capability groups are the
// UNION (content adds video/production/editing ops; outreach adds the composed
// `notify` command with sendNotification subject+message preference and the
// 16-column notification_log.md row). Either side's API-key env var works.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	providerDefaultDiscoveryURL = "https://widecast.ai/openapi.yaml"
	providerDefaultWidecastURL  = "https://widecast.ai/app/dashboard"
	providerUserAgent           = "SoloAgencyOpenAPIAdapter/1.0"
)

var providerDefaultDisabled = map[string][]string{
	"widecast": {"https://api.widecast.ai"},
}

var providerHTTPMethods = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}

// Union of both forks' candidate maps.
var knownOperationCandidates = map[string][]string{
	"account":            {"getAccount"},
	"analytics":          {"getAnalytics"},
	"list_videos":        {"listVideos"},
	"upload_asset":       {"uploadAsset"},
	"upload_html_report": {"uploadAsset"},
	"send_notification":  {"sendNotification", "sendTelegramMessage"},
	"publish":            {"publish"},
	"create_video":       {"createVideo"},
	"export_video":       {"exportVideo"},
	"get_status":         {"getStatus", "waitForVideo"},
	"get_video_data":     {"getVideoData", "videoData"},
	"get_writing_skill":  {"getWritingSkill"},
	"get_editing_skill":  {"getEditingSkill"},
	"create_content":     {"createContent"},
	"create_image":       {"createImage"},
	"search_broll":       {"searchBroll"},
	"collect_ideas":      {"collectIdeas"},
	"scene_geometry":     {"sceneGeometry", "getSceneGeometry"},
	"scene_inspector":    {"sceneInspector", "inspectScene", "getSceneInspector"},
	"modify_scene":       {"modifyScene"},
}

// Union of both forks' capability groups (content superset + outreach analytics).
var capabilityGroupAliases = map[string][]string{
	"production":    {"create_video", "get_status"},
	"video_editing": {"get_editing_skill", "get_video_data", "scene_geometry", "scene_inspector", "modify_scene"},
	"render_export": {"export_video"},
	"media":         {"upload_asset", "create_image", "search_broll"},
	"distribution":  {"publish"},
	"notification":  {"send_notification"},
	"analytics":     {"account", "analytics"},
}

type providerError struct{ msg string }

func (e *providerError) Error() string { return e.msg }

func providerErrf(format string, a ...any) error {
	return &providerError{msg: fmt.Sprintf(format, a...)}
}

// --- config -----------------------------------------------------------------------

func providerReadJSON(path string) (map[string]any, error) {
	if path == "" {
		return map[string]any{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, providerErrf("provider_config_unreadable: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, providerErrf("provider_config_unreadable: %v", err)
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func providerBlock(config map[string]any, provider string) map[string]any {
	return orEmptyMap(mMap(mMap(config, "providers"), provider))
}

func activeProvider(config map[string]any, fallback string) string {
	if v := mStr(config, "active_provider"); v != "" {
		return v
	}
	return fallback
}

func providerAPIKey(config map[string]any, provider string) (string, error) {
	block := providerBlock(config, provider)
	for _, env := range []string{"OUTREACHCRM_PROVIDER_API_KEY", "SOLO_AGENCY_PROVIDER_API_KEY"} {
		if v := os.Getenv(env); v != "" {
			return v, nil
		}
	}
	if envName := mStr(block, "api_key_env"); envName != "" {
		if v := os.Getenv(envName); v != "" {
			return v, nil
		}
	}
	if local := block["api_key_local"]; truthy(local) {
		return fmt.Sprint(local), nil
	}
	return "", providerErrf("provider_auth_missing: no api_key_env value resolved and api_key_local is empty")
}

func normalizeServerURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

func asStringList(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, item := range x {
			s := fmt.Sprint(item)
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func providerDiscoveryURL(defaults, config map[string]any, provider string) string {
	if v := mStr(providerBlock(config, provider), "discovery_url"); v != "" {
		return v
	}
	if v := mStr(mMap(mMap(defaults, "providers"), provider), "discovery_url"); v != "" {
		return v
	}
	return providerDefaultDiscoveryURL
}

func disabledServerURLs(defaults, config map[string]any, provider string) map[string]bool {
	out := map[string]bool{}
	for _, u := range providerDefaultDisabled[provider] {
		out[normalizeServerURL(u)] = true
	}
	for _, src := range []any{
		mMap(mMap(defaults, "providers"), provider)["disabled_server_urls"],
		providerBlock(config, provider)["disabled_server_urls"],
	} {
		for _, u := range asStringList(src) {
			if n := normalizeServerURL(u); n != "" {
				out[n] = true
			}
		}
	}
	delete(out, "")
	return out
}

func preferredServerURLs(defaults, config map[string]any, provider string) []string {
	defaultBlock := mMap(mMap(defaults, "providers"), provider)
	block := providerBlock(config, provider)
	var preferred []string
	for _, src := range []any{block["server_url"], block["preferred_server_url"],
		defaultBlock["server_url"], defaultBlock["preferred_server_url"]} {
		preferred = append(preferred, asStringList(src)...)
	}
	if provider == "widecast" {
		preferred = append(preferred, providerDefaultWidecastURL)
	}
	seen := map[string]bool{}
	var result []string
	for _, u := range preferred {
		n := normalizeServerURL(u)
		if n != "" && !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	return result
}

func selectServerURL(defaults, config map[string]any, provider string, serverURLs []string) (string, []string, error) {
	disabled := disabledServerURLs(defaults, config, provider)
	var cleaned []string
	seen := map[string]bool{}
	for _, u := range serverURLs {
		n := normalizeServerURL(u)
		if n != "" && !seen[n] {
			seen[n] = true
			cleaned = append(cleaned, n)
		}
	}
	var skipped []string
	for _, u := range cleaned {
		if disabled[u] {
			skipped = append(skipped, u)
		}
	}
	for _, u := range preferredServerURLs(defaults, config, provider) {
		if disabled[u] {
			found := false
			for _, s := range skipped {
				if s == u {
					found = true
					break
				}
			}
			if !found {
				skipped = append(skipped, u)
			}
			continue
		}
		return u, skipped, nil
	}
	for _, u := range cleaned {
		if disabled[u] {
			continue
		}
		return u, skipped, nil
	}
	if len(skipped) > 0 {
		return "", skipped, providerErrf("provider_discovery_failed: all discovered/preferred OpenAPI server URLs are disabled: %s",
			strings.Join(skipped, ", "))
	}
	return "", skipped, providerErrf("provider_discovery_failed: OpenAPI server URL not found")
}

// --- HTTP -------------------------------------------------------------------------

var providerHTTPClient = &http.Client{Timeout: 90 * time.Second}

func providerFetch(fetchURL string) (int, string, []byte, error) {
	req, err := http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		return 0, "", nil, providerErrf("network_error: %v", err)
	}
	req.Header.Set("User-Agent", providerUserAgent)
	req.Header.Set("Accept", "application/yaml, application/json, text/yaml, */*")
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", nil, providerErrf("network_error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		b := body
		if len(b) > 500 {
			b = b[:500]
		}
		return resp.StatusCode, "", nil, providerErrf("http_error: status=%d body=%s", resp.StatusCode, string(b))
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func providerRequestJSON(method, requestURL, apiKey string, body map[string]any) (int, http.Header, any, error) {
	var reader io.Reader
	req, err := http.NewRequest(strings.ToUpper(method), requestURL, nil)
	if err != nil {
		return 0, nil, nil, providerErrf("network_error: %v", err)
	}
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, nil, nil, err
		}
		reader = strings.NewReader(string(payload))
		req, err = http.NewRequest(strings.ToUpper(method), requestURL, reader)
		if err != nil {
			return 0, nil, nil, providerErrf("network_error: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", providerUserAgent)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, nil, providerErrf("network_error: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	text := string(raw)
	var parsed any
	if text == "" {
		parsed = map[string]any{}
	} else if json.Unmarshal(raw, &parsed) != nil {
		parsed = map[string]any{"raw": text}
	}
	return resp.StatusCode, resp.Header, parsed, nil
}

// --- OpenAPI parsing (JSON + hand-rolled YAML line scanner) -----------------------

type openAPISpec struct {
	ServerURLs []string
	ServerURL  string
	Operations map[string]map[string]string // opId -> {method, path}
	Skipped    []string
}

var (
	yamlPathRe   = regexp.MustCompile(`^  (/[^\s:]+):\s*$`)
	yamlMethodRe = regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
	yamlOpIDRe   = regexp.MustCompile(`^\s+operationId:\s*([A-Za-z0-9_:-]+)\s*$`)
)

func parseOpenAPIYAML(text string) (*openAPISpec, error) {
	var serverURLs []string
	operations := map[string]map[string]string{}
	currentPath, currentMethod := "", ""
	inServers, inPaths := false, false
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		stripped := strings.TrimSpace(line)
		if stripped == "servers:" {
			inServers = true
			continue
		}
		if inServers && strings.HasPrefix(stripped, "- url:") {
			v := strings.TrimSpace(strings.SplitN(stripped, ":", 2)[1])
			v = strings.Trim(v, `"'`)
			serverURLs = append(serverURLs, v)
			continue
		}
		if stripped == "paths:" {
			inPaths = true
			inServers = false
			continue
		}
		if !inPaths {
			continue
		}
		if m := yamlPathRe.FindStringSubmatch(line); m != nil {
			currentPath = m[1]
			currentMethod = ""
			continue
		}
		if m := yamlMethodRe.FindStringSubmatch(line); m != nil {
			currentMethod = m[1]
			continue
		}
		if m := yamlOpIDRe.FindStringSubmatch(line); m != nil && currentPath != "" && currentMethod != "" {
			opID := strings.Trim(strings.TrimSpace(m[1]), `"'`)
			operations[opID] = map[string]string{"method": strings.ToUpper(currentMethod), "path": currentPath}
		}
	}
	if len(serverURLs) == 0 {
		return nil, providerErrf("provider_discovery_failed: OpenAPI server URL not found")
	}
	return &openAPISpec{ServerURLs: serverURLs, ServerURL: serverURLs[0], Operations: operations}, nil
}

func parseOpenAPI(raw []byte, contentType string) (*openAPISpec, string, error) {
	text := string(raw)
	if strings.Contains(contentType, "json") || strings.HasPrefix(strings.TrimLeft(text, " \t\r\n"), "{") {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, text, err
		}
		var serverURLs []string
		for _, item := range mapsOf(mList(doc, "servers")) {
			if u := mStr(item, "url"); u != "" {
				serverURLs = append(serverURLs, u)
			}
		}
		operations := map[string]map[string]string{}
		for path, methodsAny := range mMap(doc, "paths") {
			methods, ok := methodsAny.(map[string]any)
			if !ok {
				continue
			}
			for method, opAny := range methods {
				if !providerHTTPMethods[strings.ToLower(method)] {
					continue
				}
				op, ok := opAny.(map[string]any)
				if !ok {
					continue
				}
				if opID := mStr(op, "operationId"); opID != "" {
					operations[opID] = map[string]string{"method": strings.ToUpper(method), "path": path}
				}
			}
		}
		if len(serverURLs) == 0 {
			return nil, text, providerErrf("provider_discovery_failed: OpenAPI server URL not found")
		}
		return &openAPISpec{ServerURLs: serverURLs, ServerURL: serverURLs[0], Operations: operations}, text, nil
	}
	spec, err := parseOpenAPIYAML(text)
	return spec, text, err
}

func loadSpec(discoveryURLFlag, providerFlag string, config, defaults map[string]any) (string, *openAPISpec, string, error) {
	provider := activeProvider(config, providerFlag)
	u := discoveryURLFlag
	if u == "" {
		u = providerDiscoveryURL(defaults, config, provider)
	}
	status, contentType, raw, err := providerFetch(u)
	if err != nil {
		return provider, nil, "", err
	}
	if status >= 400 {
		return provider, nil, "", providerErrf("provider_discovery_failed: status=%d", status)
	}
	spec, rawText, err := parseOpenAPI(raw, contentType)
	if err != nil {
		return provider, nil, rawText, err
	}
	candidates := spec.ServerURLs
	if len(candidates) == 0 {
		candidates = []string{spec.ServerURL}
	}
	selected, skipped, err := selectServerURL(defaults, config, provider, candidates)
	if err != nil {
		return provider, nil, rawText, err
	}
	spec.ServerURL = selected
	spec.Skipped = skipped
	return provider, spec, rawText, nil
}

func operationAliases(operations map[string]map[string]string) map[string]string {
	byLower := map[string]string{}
	for opID := range operations {
		byLower[strings.ToLower(opID)] = opID
	}
	aliases := map[string]string{}
	for alias, candidates := range knownOperationCandidates {
		for _, cand := range candidates {
			if opID, ok := byLower[strings.ToLower(cand)]; ok {
				aliases[alias] = opID
				break
			}
		}
	}
	return aliases
}

func capabilityStatus(aliases map[string]string) map[string]any {
	status := map[string]any{}
	for group, required := range capabilityGroupAliases {
		present := 0
		for _, a := range required {
			if _, ok := aliases[a]; ok {
				present++
			}
		}
		switch {
		case present == len(required):
			status[group] = "available"
		case present > 0:
			status[group] = "partial"
		default:
			status[group] = "unavailable"
		}
	}
	return status
}

func missingCapabilityAliases(aliases map[string]string) map[string]any {
	missing := map[string]any{}
	for group, required := range capabilityGroupAliases {
		var groupMissing []any
		for _, a := range required {
			if _, ok := aliases[a]; !ok {
				groupMissing = append(groupMissing, a)
			}
		}
		if len(groupMissing) > 0 {
			missing[group] = groupMissing
		}
	}
	return missing
}

func urlFor(serverURL, path string, query []string) (string, error) {
	u := strings.TrimRight(serverURL, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		vals := url.Values{}
		var order []string
		for _, item := range query {
			k, v, ok := strings.Cut(item, "=")
			if !ok {
				return "", providerErrf("bad_query: expected key=value, got %s", item)
			}
			vals.Add(k, v)
			order = append(order, k)
		}
		_ = order
		u += "?" + vals.Encode()
	}
	return u, nil
}

func writeCallLog(configPath, provider, operation string, status int, blocker string) {
	if configPath == "" {
		return
	}
	logPath := filepath.Join(filepath.Dir(configPath), "provider_calls.jsonl")
	rec := map[string]any{
		"ts": time.Now().UTC().Format("2006-01-02T15:04:05Z"), "provider": provider,
		"operationId": operation, "status": status, "blocker": blocker,
	}
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		f.WriteString(marshalLineJSON(rec) + "\n")
		f.Close()
	}
}

func selectedHeaders(h http.Header) map[string]any {
	keep := map[string]any{}
	for key, vals := range h {
		lk := strings.ToLower(key)
		switch lk {
		case "x-request-id", "x-ratelimit-limit", "x-ratelimit-remaining", "x-ratelimit-reset", "x-widecast-version":
			if len(vals) > 0 {
				keep[key] = vals[0]
			}
		}
	}
	return keep
}

// --- commands ---------------------------------------------------------------------

type providerArgs struct {
	Config       string
	Defaults     string
	Provider     string
	DiscoveryURL string
}

func providerCmdDiscover(pa providerArgs, outDir string) (int, error) {
	config, err := providerReadJSON(pa.Config)
	if err != nil {
		return 2, err
	}
	defaults, err := providerReadJSON(pa.Defaults)
	if err != nil {
		return 2, err
	}
	provider, spec, rawText, err := loadSpec(pa.DiscoveryURL, pa.Provider, config, defaults)
	if err != nil {
		return 2, err
	}
	aliases := operationAliases(spec.Operations)
	sortedOps := map[string]any{}
	for k, v := range spec.Operations {
		sortedOps[k] = map[string]any{"method": v["method"], "path": v["path"]}
	}
	sortedAliases := map[string]any{}
	for k, v := range aliases {
		sortedAliases[k] = v
	}
	discoveryURL := pa.DiscoveryURL
	if discoveryURL == "" {
		discoveryURL = providerDiscoveryURL(defaults, config, provider)
	}
	out := map[string]any{
		"schema_version": 1, "provider": provider,
		"discovered_at": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"discovery_url": discoveryURL, "server_url": spec.ServerURL,
		"server_urls_discovered":       strsToAny(spec.ServerURLs),
		"server_urls_skipped_disabled": strsToAny(spec.Skipped),
		"operation_ids":                sortedOps,
		"operation_aliases":            sortedAliases,
		"capability_status":            capabilityStatus(aliases),
		"missing_capability_aliases":   missingCapabilityAliases(aliases),
	}
	if outDir != "" {
		if err := writeRenderText(filepath.Join(outDir, "provider_capabilities.json"), marshalIndentJSON(out)+"\n"); err != nil {
			return 2, err
		}
		if err := writeRenderText(filepath.Join(outDir, "provider_openapi_cache.yaml"), rawText); err != nil {
			return 2, err
		}
	}
	fmt.Println(marshalIndentJSON(out))
	return 0, nil
}

func providerCmdCall(pa providerArgs, operation, bodyJSON string, query []string, noAuth bool) (int, error) {
	config, err := providerReadJSON(pa.Config)
	if err != nil {
		return 2, err
	}
	defaults, err := providerReadJSON(pa.Defaults)
	if err != nil {
		return 2, err
	}
	provider, spec, _, err := loadSpec(pa.DiscoveryURL, pa.Provider, config, defaults)
	if err != nil {
		return 2, err
	}
	op, ok := spec.Operations[operation]
	if !ok {
		return 2, providerErrf("provider_required_operation_missing: %s", operation)
	}
	var body map[string]any
	if bodyJSON != "" {
		if err := json.Unmarshal([]byte(bodyJSON), &body); err != nil {
			return 2, err
		}
	}
	callURL, err := urlFor(spec.ServerURL, op["path"], query)
	if err != nil {
		return 2, err
	}
	key := ""
	if !noAuth {
		key, err = providerAPIKey(config, provider)
		if err != nil {
			return 2, err
		}
	}
	status, headers, response, err := providerRequestJSON(op["method"], callURL, key, body)
	if err != nil {
		return 2, err
	}
	blocker := ""
	if status >= 400 {
		blocker = fmt.Sprintf("provider_call_failed: status=%d", status)
	}
	writeCallLog(pa.Config, provider, operation, status, blocker)
	fmt.Println(marshalIndentJSON(map[string]any{"status": status,
		"headers": selectedHeaders(headers), "body": response}))
	if status >= 400 {
		return 1, nil
	}
	return 0, nil
}

func providerCmdUploadReport(pa providerArgs, file, operation, contentType string, noAuth bool) (int, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return 2, providerErrf("file_not_found: %s", file)
	}
	if operation == "" {
		operation = "uploadAsset"
	}
	body := map[string]any{
		"file_data": base64.StdEncoding.EncodeToString(data),
		"filename":  filepath.Base(file), "content_type": contentType,
	}
	return providerCmdCall(pa, operation, marshalLineJSON(body), nil, noAuth)
}

// --- composed notify --------------------------------------------------------------

var notifyLogColumns = []string{
	"Date", "Agent", "Event", "Channel", "Status", "Report Path", "Report Link Sent",
	"Provider", "Provider Discovery Checked", "Upload Operation", "Notification Operation",
	"Upload Attempted", "Uploaded Report URL", "Notification Attempted", "Blocker", "Action Needed",
}

func appendNotificationLog(logPath string, row map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	existing, _ := os.ReadFile(logPath)
	if strings.TrimSpace(string(existing)) == "" {
		header := "# Notification Log\n\n| " + strings.Join(notifyLogColumns, " | ") + " |\n" +
			"|" + strings.Repeat("---|", len(notifyLogColumns)) + "\n"
		if err := os.WriteFile(logPath, []byte(header), 0o644); err != nil {
			return err
		}
	}
	cells := make([]string, len(notifyLogColumns))
	for i, c := range notifyLogColumns {
		v := row[c]
		v = strings.ReplaceAll(v, "|", `\|`)
		v = strings.ReplaceAll(v, "\n", " ")
		cells[i] = v
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	return err
}

type notifyArgs struct {
	providerArgs
	Message      string
	Subject      string
	Event        string
	ReportFile   string
	ReportPath   string
	Log          string
	Agent        string
	ActionNeeded string
	DryRun       bool
}

func providerCmdNotify(na notifyArgs) (int, error) {
	config, err := providerReadJSON(na.Config)
	if err != nil {
		return 2, err
	}
	defaults, err := providerReadJSON(na.Defaults)
	if err != nil {
		return 2, err
	}
	provider := activeProvider(config, na.Provider)
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	reportPath := na.ReportPath
	if reportPath == "" {
		reportPath = na.ReportFile
	}
	row := map[string]string{"Date": now[:10], "Agent": na.Agent, "Event": na.Event,
		"Channel": "WideCast email+Telegram", "Provider": provider,
		"Report Path": reportPath, "Action Needed": na.ActionNeeded,
		"Provider Discovery Checked": "no", "Upload Attempted": "no",
		"Notification Attempted": "no", "Report Link Sent": "no"}

	finish := func(status, blocker string, exitCode int, extra map[string]string) (int, error) {
		row["Status"] = status
		if blocker == "" {
			row["Blocker"] = "none"
		} else {
			row["Blocker"] = blocker
		}
		for k, v := range extra {
			row[k] = v
		}
		if na.Log != "" {
			if err := appendNotificationLog(na.Log, row); err != nil {
				return 1, err
			}
		}
		out := map[string]any{"status": status, "blocker": nil, "provider": provider,
			"event": na.Event, "dry_run": na.DryRun}
		if blocker != "" {
			out["blocker"] = blocker
		}
		for k, v := range extra {
			out[strings.ReplaceAll(strings.ToLower(k), " ", "_")] = v
		}
		fmt.Println(marshalIndentJSON(out))
		return exitCode, nil
	}

	notifCfg := orEmptyMap(mMap(providerBlock(config, provider), "notification"))
	if len(config) == 0 || len(providerBlock(config, provider)) == 0 {
		return finish("local_path_only", "provider_config_missing", 0, nil)
	}
	if !truthy(notifCfg["enabled"]) {
		return finish("local_path_only", "provider_notification_not_configured", 0, nil)
	}
	key, keyErr := providerAPIKey(config, provider)
	if keyErr != nil {
		return finish("local_path_only", "provider_auth_missing", 0, nil)
	}

	if na.DryRun {
		extra := map[string]string{"Notification Operation": "sendNotification (fallback: sendTelegramMessage)"}
		if na.ReportFile != "" {
			extra["Upload Operation"] = "uploadAsset"
		} else {
			extra["Upload Operation"] = ""
		}
		return finish("dry_run", "", 0, extra)
	}

	_, spec, _, err := loadSpec(na.DiscoveryURL, na.Provider, config, defaults)
	if err != nil {
		blocker := strings.SplitN(err.Error(), ":", 2)[0]
		if blocker == "" {
			blocker = "provider_discovery_failed"
		}
		return finish("blocked", blocker, 1, nil)
	}
	row["Provider Discovery Checked"] = "yes"
	aliases := operationAliases(spec.Operations)
	if _, ok := aliases["send_notification"]; !ok {
		return finish("blocked", "provider_required_operation_missing", 1, nil)
	}
	row["Notification Operation"] = aliases["send_notification"]

	uploadCfg := orEmptyMap(mMap(providerBlock(config, provider), "report_upload"))
	urlFields := asStringList(uploadCfg["url_fields"])
	if len(urlFields) == 0 {
		urlFields = []string{"url", "asset_url", "link", "download_url"}
	}
	uploadedURL := ""
	softBlocker := ""
	if na.ReportFile != "" {
		if up, ok := aliases["upload_asset"]; ok {
			row["Upload Operation"] = up
		} else {
			row["Upload Operation"] = "uploadAsset"
		}
		if _, ok := aliases["upload_asset"]; !ok {
			softBlocker = "provider_upload_operation_missing"
		} else if _, err := os.Stat(na.ReportFile); err != nil {
			softBlocker = "provider_upload_failed"
		} else {
			row["Upload Attempted"] = "yes"
			up := spec.Operations[aliases["upload_asset"]]
			data, _ := os.ReadFile(na.ReportFile)
			body := map[string]any{"file_data": base64.StdEncoding.EncodeToString(data),
				"filename": filepath.Base(na.ReportFile), "content_type": "text/html"}
			upURL, _ := urlFor(spec.ServerURL, up["path"], nil)
			st, _, resp, err := providerRequestJSON(up["method"], upURL, key, body)
			if err != nil || st >= 400 {
				softBlocker = "provider_upload_failed"
			} else if respMap, ok := resp.(map[string]any); ok {
				for _, k := range urlFields {
					if truthy(respMap[k]) {
						uploadedURL = fmt.Sprint(respMap[k])
						break
					}
				}
				if uploadedURL == "" {
					softBlocker = "provider_upload_url_unrecognized"
				}
			}
		}
		if softBlocker != "" {
			row["Blocker"] = softBlocker
		}
	}

	opID := aliases["send_notification"]
	op := spec.Operations[opID]
	text := na.Message
	if uploadedURL != "" {
		text += "\n\nReport: " + uploadedURL
	}
	row["Notification Attempted"] = "yes"
	var body map[string]any
	if opID == "sendNotification" || mStr(notifCfg, "subject_field") != "" {
		subject := na.Subject
		if subject == "" {
			lines := strings.Split(strings.TrimSpace(na.Message), "\n")
			first := "Solo Agency report"
			if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
				first = lines[0]
			}
			if len(first) > 120 {
				first = first[:120]
			}
			subject = first
		}
		subjectField := mStr(notifCfg, "subject_field")
		if subjectField == "" {
			subjectField = "subject"
		}
		messageField := mStr(notifCfg, "message_field")
		if messageField == "" {
			messageField = "message"
		}
		body = map[string]any{subjectField: subject, messageField: text}
	} else {
		textField := mStr(notifCfg, "text_field")
		if textField == "" {
			textField = "text"
		}
		body = map[string]any{textField: text}
	}
	sendURL, _ := urlFor(spec.ServerURL, op["path"], nil)
	st, _, _, err := providerRequestJSON(op["method"], sendURL, key, body)
	callBlocker := ""
	if err != nil || st >= 400 {
		callBlocker = "provider_notification_failed"
	}
	writeCallLog(na.Config, provider, opID, st, callBlocker)
	linkSent := "no"
	if uploadedURL != "" {
		linkSent = "yes"
	}
	extra := map[string]string{"Uploaded Report URL": uploadedURL, "Report Link Sent": linkSent}
	if callBlocker != "" {
		return finish("blocked", "provider_notification_failed", 1, extra)
	}
	return finish("sent", softBlocker, 0, extra)
}

// --- CLI: tool provider -----------------------------------------------------------

func runProviderCLI(args []string) int {
	valueFlags := map[string]bool{"--config": true, "--defaults": true, "--provider": true,
		"--discovery-url": true, "--out-dir": true, "--operation": true, "--body": true,
		"--query": true, "--file": true, "--content-type": true, "--message": true,
		"--subject": true, "--event": true, "--report-file": true, "--report-path": true,
		"--log": true, "--agent": true, "--action-needed": true}
	boolFlags := map[string]bool{"--no-auth": true, "--dry-run": true}
	a, err := parseCLIArgs(args, valueFlags, boolFlags)
	if err != nil {
		return crmUsageErr(err.Error())
	}
	if len(a.pos) == 0 {
		return crmUsageErr("a subcommand is required (discover | call | account | upload-report | notify)")
	}
	pa := providerArgs{Config: a.get("--config"), Defaults: a.get("--defaults"),
		Provider: strOr(a.get("--provider"), "widecast"), DiscoveryURL: a.get("--discovery-url")}
	fail := func(rc int, err error) int {
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		return rc
	}
	switch a.pos[0] {
	case "discover":
		rc, err := providerCmdDiscover(pa, a.get("--out-dir"))
		return fail(rc, err)
	case "call":
		if a.get("--operation") == "" {
			return crmUsageErr("call needs --operation")
		}
		rc, err := providerCmdCall(pa, a.get("--operation"), a.get("--body"), a.flags["--query"], a.bools["--no-auth"])
		return fail(rc, err)
	case "account":
		rc, err := providerCmdCall(pa, strOr(a.get("--operation"), "getAccount"), "", nil, a.bools["--no-auth"])
		return fail(rc, err)
	case "upload-report":
		if a.get("--file") == "" {
			return crmUsageErr("upload-report needs --file")
		}
		rc, err := providerCmdUploadReport(pa, a.get("--file"), a.get("--operation"),
			strOr(a.get("--content-type"), "text/html"), a.bools["--no-auth"])
		return fail(rc, err)
	case "notify":
		if a.get("--message") == "" {
			return crmUsageErr("notify needs --message")
		}
		event := strOr(a.get("--event"), "daily_run_completed")
		if event != "daily_run_completed" && event != "weekly_client_report_ready" {
			return crmUsageErr("notify --event must be daily_run_completed | weekly_client_report_ready")
		}
		rc, err := providerCmdNotify(notifyArgs{providerArgs: pa,
			Message: a.get("--message"), Subject: a.get("--subject"), Event: event,
			ReportFile: a.get("--report-file"), ReportPath: a.get("--report-path"),
			Log: a.get("--log"), Agent: strOr(a.get("--agent"), "Claude Schedule"),
			ActionNeeded: a.get("--action-needed"), DryRun: a.bools["--dry-run"]})
		return fail(rc, err)
	}
	return crmUsageErr("unknown provider subcommand " + a.pos[0])
}

var _ = sort.Strings
