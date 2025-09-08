package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// BuildAffiliateURL will replace all placeholders {key} in baseURL with queryParams[key] if present,
// otherwise fallback ke sub_id, lalu tambahin extra query yg ga ada di template.
func BuildAffiliateURL(baseURL string, queryParams map[string]string) string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(baseURL, -1)

	mainSub := queryParams["sub_id"]

	var replacerPairs []string
	for _, m := range matches {
		placeholder := m[0] // e.g. {sub_id_1}
		key := m[1]         // e.g. sub_id_1
		val, ok := queryParams[key]
		if (!ok || val == "") && key == "siteid" {
			// Support both alias styles: sub_id_1 and sub_id1
			if alt, ok2 := queryParams["sub_id_1"]; ok2 && alt != "" {
				val = alt
				ok = true
			} else if alt2, ok3 := queryParams["sub_id1"]; ok3 && alt2 != "" {
				val = alt2
				ok = true
			}
		}
		if !ok || val == "" {
			// if the placeholder is type_ads and not provided, skip replacing with main sub_id
			if key == "type_ads" {
				continue
			}
			// if the placeholder is siteid and not provided (or alias not found), do not default
			if key == "siteid" {
				continue
			}
			// also if key looks like sub_id1/sub_id_1 etc and not provided, skip defaulting unless explicitly present
			if strings.HasPrefix(key, "sub_id") && key != "sub_id" {
				continue
			}
			val = mainSub
		}
		replacerPairs = append(replacerPairs, placeholder, url.QueryEscape(val))
	}

	replacer := strings.NewReplacer(replacerPairs...)
	finalURL := replacer.Replace(baseURL)

	// remove unresolved placeholders from query (e.g., sub_id1={siteid} when siteid absent)
	finalURL = removeUnresolvedParams(finalURL)

	var extra []string
	// ensure deterministic order and avoid duplicating params already present after replacement
	present := map[string]struct{}{}
	{
		parts := strings.SplitN(finalURL, "?", 2)
		if len(parts) == 2 {
			for _, it := range strings.Split(parts[1], "&") {
				if it == "" {
					continue
				}
				kv := strings.SplitN(it, "=", 2)
				if len(kv) >= 1 && kv[0] != "" {
					present[kv[0]] = struct{}{}
				}
			}
		}
	}
	keys := make([]string, 0, len(queryParams))
	for k := range queryParams {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		v := queryParams[k]
		if strings.Contains(baseURL, "{"+k+"}") { // ada di template
			continue
		}
		if _, exists := present[url.QueryEscape(k)]; exists { // sudah ada di final URL
			continue
		}
		extra = append(extra, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
	}

	if len(extra) > 0 {
		sep := "?"
		if strings.Contains(finalURL, "?") {
			sep = "&"
		}
		finalURL += sep + strings.Join(extra, "&")
	}

	return finalURL
}

// removeUnresolvedParams removes any query parameters whose values still contain
// unresolved placeholder braces (e.g., {type_ads}) or are empty.
func removeUnresolvedParams(u string) string {
	parts := strings.SplitN(u, "?", 2)
	if len(parts) < 2 {
		return u
	}
	base := parts[0]
	qs := parts[1]
	if qs == "" {
		return u
	}
	items := strings.Split(qs, "&")
	var kept []string
	for _, it := range items {
		if it == "" {
			continue
		}
		kv := strings.SplitN(it, "=", 2)
		if len(kv) != 2 {
			continue
		}
		val := kv[1]
		if val == "" {
			continue
		}
		if strings.Contains(val, "{") || strings.Contains(val, "}") {
			continue
		}
		kept = append(kept, it)
	}
	if len(kept) == 0 {
		return base
	}
	return base + "?" + strings.Join(kept, "&")
}
