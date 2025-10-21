package validation

import "github.com/microcosm-cc/bluemonday"

var strict_policy *bluemonday.Policy
var pre_policy *bluemonday.Policy

func initXSS() {
	strict_policy = bluemonday.StrictPolicy()
	pre_policy = bluemonday.UGCPolicy()
}

func StrictSanitize(text string) string {
	return strict_policy.Sanitize(text)
}

func PreRenderSanitization(text string) string {
	return pre_policy.Sanitize(text)
}
