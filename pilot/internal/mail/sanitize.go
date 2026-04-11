package mail

import "regexp"

// SanitizeHTML is a defense-in-depth regex-based sanitizer for HTML email
// bodies submitted via the send API. It is NOT a complete HTML parser —
// it simply strips the most dangerous constructs:
//
//   - <script>...</script> blocks (case-insensitive)
//   - <iframe>, <object>, <embed>, <applet> tags
//   - javascript: and vbscript: URL schemes
//   - data:text/html URLs
//   - on* event handlers (onclick, onerror, onload, ...)
//
// For robust sanitization (context-aware parsing, CSS, SVG), consider
// using a proper HTML sanitizer library like bluemonday.
//
// This function is safe to call on empty strings — it returns "".
func SanitizeHTML(s string) string {
	if s == "" {
		return s
	}
	for _, re := range sanitizePatterns {
		s = re.ReplaceAllString(s, "")
	}
	// Neutralize dangerous URL schemes in attributes (href, src, action, ...)
	s = schemeRe.ReplaceAllString(s, `$1="#"`)
	return s
}

var (
	// Full tag block strippers (case-insensitive, multiline).
	sanitizePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>`),
		regexp.MustCompile(`(?is)<script\b[^>]*/?>`),
		regexp.MustCompile(`(?is)<iframe\b[^>]*>.*?</iframe\s*>`),
		regexp.MustCompile(`(?is)<iframe\b[^>]*/?>`),
		regexp.MustCompile(`(?is)<object\b[^>]*>.*?</object\s*>`),
		regexp.MustCompile(`(?is)<embed\b[^>]*/?>`),
		regexp.MustCompile(`(?is)<applet\b[^>]*>.*?</applet\s*>`),
		regexp.MustCompile(`(?is)<link\b[^>]*rel\s*=\s*["']?import["']?[^>]*/?>`),
		// Strip inline event handlers: on<word>=<"...">, on<word>='...', on<word>=value
		regexp.MustCompile(`(?i)\s+on[a-z]+\s*=\s*"[^"]*"`),
		regexp.MustCompile(`(?i)\s+on[a-z]+\s*=\s*'[^']*'`),
		regexp.MustCompile(`(?i)\s+on[a-z]+\s*=\s*[^\s>]+`),
	}

	// Neutralize dangerous URL schemes in attributes. Matches
	// href="javascript:...", src='vbscript:...', data:text/html URIs, etc.
	schemeRe = regexp.MustCompile(`(?i)\b(href|src|action|formaction|xlink:href)\s*=\s*["']?\s*(?:javascript|vbscript|data:text/html)[^"'\s>]*["']?`)
)
