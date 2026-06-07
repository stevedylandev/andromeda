package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
	"regexp"
)

type parsedAttributes struct {
	Title           string
	Slug            string
	Alias           string
	PublishedDate   string
	MetaDescription string
	MetaImage       string
	Lang            string
	Tags            string
	Status          string
	Weather         string
}

type WeatherPointResponse struct {
	Properties struct {
		ForecastHourly   string `json:"forecastHourly"`
		RelativeLocation struct {
			Properties struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"properties"`
		} `json:"relativeLocation"`
	} `json:"properties"`
}

type Period struct {
	Temperature   int    `json:"temperature"`
	ShortForecast string `json:"shortForecast"`
}

type WeatherForecastResponse struct {
	Properties struct {
		Periods []Period `json:"periods"`
	} `json:"properties"`
}

type Weather struct {
	Icon        template.HTML
	Conditions  string
	Temperature string
	Location    string
}

func parseAttributes(text string) parsedAttributes {
	var a parsedAttributes
	for _, line := range strings.Split(text, "\n") {
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		value := strings.TrimSpace(line[i+1:])
		switch key {
		case "title":
			a.Title = value
		case "slug":
			a.Slug = value
		case "alias":
			a.Alias = value
		case "published_date":
			a.PublishedDate = value
		case "description", "meta_description":
			a.MetaDescription = value
		case "meta_image":
			a.MetaImage = value
		case "lang":
			a.Lang = value
		case "tags":
			a.Tags = value
		case "status":
			a.Status = value
		case "weather":
			a.Weather = value
		}
	}
	return a
}

type parsedPageAttributes struct {
	Title       string
	Slug        string
	IsPublished bool
}

func parsePageAttributes(text string) parsedPageAttributes {
	var a parsedPageAttributes
	for _, line := range strings.Split(text, "\n") {
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		value := strings.TrimSpace(line[i+1:])
		switch key {
		case "title":
			a.Title = value
		case "slug":
			a.Slug = value
		case "published":
			a.IsPublished = value == "true"
		}
	}
	return a
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	parts := strings.Split(b.String(), "-")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "-")
}

func optStr(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}

func deriveSlug(title, slug string) string {
	if slug != "" {
		return slug
	}
	if from := slugify(title); from != "" {
		return from
	}
	id, _ := generateID()
	return id
}

func generateID() (string, error) {
	// Imported from auth crate at call site; this is a fallback path.
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, 10)
	for i := range buf {
		buf[i] = alphabet[time.Now().UnixNano()%int64(len(alphabet))]
		time.Sleep(time.Nanosecond)
	}
	return string(buf), nil
}

var reservedPageSlugs = map[string]bool{
	"posts": true, "admin": true, "feed.xml": true,
	"custom-styles.css": true, "static": true, "files": true,
}

func isReservedPageSlug(slug string) bool {
	return reservedPageSlugs[slug]
}

func parseNavLinks(input string) []NavLink {
	var out []NavLink
	rest := input
	for {
		open := strings.Index(rest, "[")
		if open < 0 {
			break
		}
		close := strings.Index(rest[open:], "]")
		if close < 0 {
			break
		}
		close += open
		label := rest[open+1 : close]
		if close+1 >= len(rest) || rest[close+1] != '(' {
			rest = rest[close+1:]
			continue
		}
		urlEnd := strings.Index(rest[close+2:], ")")
		if urlEnd < 0 {
			break
		}
		urlEnd += close + 2
		url := rest[close+2 : urlEnd]
		if label != "" && url != "" {
			out = append(out, NavLink{Label: label, URL: url})
		}
		rest = rest[urlEnd+1:]
	}
	return out
}

var pubDateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// parsePubDate accepts RFC3339, naive datetime, or date-only input and returns
// the value normalized to RFC3339 UTC. Returns ok=false if no layout matches.
func parsePubDate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	for _, l := range pubDateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC().Format(time.RFC3339), true
		}
	}
	return "", false
}

func toRFC2822(ts string) string {
	for _, l := range pubDateLayouts {
		if t, err := time.Parse(l, ts); err == nil {
			return t.UTC().Format(time.RFC1123Z)
		}
	}
	return ts
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

func mimeFromPath(path string) string {
	i := strings.LastIndex(path, ".")
	if i < 0 {
		return "application/octet-stream"
	}
	switch strings.ToLower(path[i+1:]) {
	case "css":
		return "text/css"
	case "js":
		return "application/javascript"
	case "html":
		return "text/html"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "ico":
		return "image/x-icon"
	case "svg":
		return "image/svg+xml"
	case "woff", "woff2":
		return "font/woff2"
	case "ttf":
		return "font/ttf"
	case "otf":
		return "font/otf"
	case "json", "webmanifest":
		return "application/json"
	case "pdf":
		return "application/pdf"
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	}
	return "application/octet-stream"
}

func postToMarkdown(p *Post) string {
	var b strings.Builder
	b.WriteString("---")
	if p.Title != nil {
		b.WriteString("\ntitle: " + *p.Title)
	}
	b.WriteString("\nslug: " + p.Slug)
	b.WriteString("\nstatus: " + p.Status)
	if p.PublishedDate != nil {
		b.WriteString("\npublished_date: " + *p.PublishedDate)
	}
	if p.Tags != nil {
		b.WriteString("\ntags: " + *p.Tags)
	}
	b.WriteString("\nlang: " + p.Lang)
	if p.Alias != nil {
		b.WriteString("\nalias: " + *p.Alias)
	}
	if p.MetaImage != nil {
		b.WriteString("\nmeta_image: " + *p.MetaImage)
	}
	if p.MetaDescription != nil {
		b.WriteString("\ndescription: " + *p.MetaDescription)
	}
	b.WriteString("\n---\n\n")
	b.WriteString(p.Content)
	return b.String()
}

func splitFrontmatter(content string) (string, string) {
	trimmed := strings.TrimPrefix(content, "\ufeff")
	var afterOpen string
	if strings.HasPrefix(trimmed, "---\n") {
		afterOpen = trimmed[4:]
	} else if strings.HasPrefix(trimmed, "---\r\n") {
		afterOpen = trimmed[5:]
	} else {
		return "", content
	}
	for _, sep := range []string{"\r\n---\r\n", "\r\n---\n", "\n---\r\n", "\n---\n"} {
		if i := strings.Index(afterOpen, sep); i >= 0 {
			body := afterOpen[i+len(sep):]
			body = strings.TrimLeft(body, "\r\n")
			return afterOpen[:i], body
		}
	}
	if strings.HasSuffix(afterOpen, "\n---") {
		return strings.TrimSuffix(afterOpen, "\n---"), ""
	}
	if strings.HasSuffix(afterOpen, "\r\n---") {
		return strings.TrimSuffix(afterOpen, "\r\n---"), ""
	}
	return "", content
}

func titleFromFilename(name string) string {
	stem := name
	if i := strings.LastIndex(name, "."); i > 0 {
		stem = name[:i]
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' {
			return ' '
		}
		return r
	}, stem)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	return strings.ToUpper(cleaned[:1]) + cleaned[1:]
}

func getWeather(location string) string {
	if location == "" {
		return ""
	}
	// Fetch Points data using lat,long
	pointURL := fmt.Sprintf("https://api.weather.gov/points/%s", location)
	resp, err := http.Get(pointURL)
	if err != nil {
		fmt.Printf("Error fetching pointUrl: %s", err.Error())
		return ""
	}
	defer resp.Body.Close()
	var weatherPoint WeatherPointResponse
	json.NewDecoder(resp.Body).Decode(&weatherPoint)
	forecastURL := weatherPoint.Properties.ForecastHourly
	city := weatherPoint.Properties.RelativeLocation.Properties.City
	state := weatherPoint.Properties.RelativeLocation.Properties.State

	// Forcast using points data
	forecastResp, err := http.Get(forecastURL)
	if err != nil {
		fmt.Printf("Error fetching forecast: %s", err.Error())
		return ""
	}
	defer resp.Body.Close()
	var weatherForecast WeatherForecastResponse
	json.NewDecoder(forecastResp.Body).Decode(&weatherForecast)
	temp := strconv.Itoa(weatherForecast.Properties.Periods[0].Temperature)
	conditions := weatherForecast.Properties.Periods[0].ShortForecast
	var qualifierRE = regexp.MustCompile(`(?i)^(slight chance|chance|isolated|scattered|patchy|areas)\s+(of\s+)?`)
	formattedConditions := strings.TrimSpace(qualifierRE.ReplaceAllString(conditions, ""))
	weather := fmt.Sprintf("%s,%s,%s,%s", formattedConditions, temp, city, state)
	return weather
}

func formatWeather(weather string) (Weather, error) {
	parts := strings.Split(weather, ",")
	if len(parts) < 4 {
		return Weather{}, fmt.Errorf("unexpected weather format: %q", weather)
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i]) // handles ", " vs ","
	}

	conditions := parts[0]
	return Weather{
		Conditions:  conditions,
		Temperature: fmt.Sprintf("%s°F", parts[1]),
		Location:    fmt.Sprintf("%s, %s", parts[2], parts[3]),
		Icon:        weatherIcons[categorize(conditions)],
	}, nil
}

type WeatherCategory int

const (
	CatStorm WeatherCategory = iota
	CatSleet
	CatSnow
	CatRain
	CatFog
	CatPartlyCloudy
	CatCloudy
	CatClear
	CatUnknown
)

var categoryRules = []struct {
	category WeatherCategory
	keywords []string
}{
	{CatStorm, []string{"thunder", "tstorm", "storm"}},
	{CatSleet, []string{"sleet", "freez", "frzg", "mix"}},
	{CatSnow, []string{"snow", "flurr"}},
	{CatRain, []string{"rain", "shower", "drizzle"}},
	{CatFog, []string{"fog", "mist", "haze"}},
	{CatPartlyCloudy, []string{"partly", "variable"}},
	{CatCloudy, []string{"overcast", "cloud"}},
	{CatClear, []string{"clear", "sunny", "sun", "fair"}},
}

func categorize(conditions string) WeatherCategory {
	c := strings.ToLower(conditions)
	for _, rule := range categoryRules {
		for _, kw := range rule.keywords {
			if strings.Contains(c, kw) {
				return rule.category
			}
		}
	}
	return CatUnknown
}

var weatherIcons = map[WeatherCategory]template.HTML{
	CatPartlyCloudy: `<svg xmlns="http://www.w3.org/2000/svg"  viewBox="0 0 256 256"><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path fill="currentColor" d="M164 76a71.9 71.9 0 0 0-22.14 3.48A51.8 51.8 0 0 0 129 63.83l11.56-16.51a4 4 0 0 0-6.56-4.59l-11.55 16.51A52 52 0 0 0 96 52c-1.71 0-3.4.09-5.06.25l-3.5-19.85a4 4 0 0 0-7.88 1.39l3.5 19.84A52.2 52.2 0 0 0 55.85 71L39.32 59.42A4 4 0 0 0 34.73 66l16.53 11.54A51.63 51.63 0 0 0 44 104c0 1.69.09 3.37.25 5l-19.85 3.5a4 4 0 0 0 .69 7.94a4 4 0 0 0 .7-.06l19.85-3.5A52.1 52.1 0 0 0 54 134.6A48 48 0 0 0 84 220h80a72 72 0 0 0 0-144M52 104a44 44 0 0 1 82.33-21.61a72.23 72.23 0 0 0-38.82 43A48.3 48.3 0 0 0 84 124a47.76 47.76 0 0 0-23.4 6.11A44 44 0 0 1 52 104m112 108H84a40 40 0 1 1 9.43-78.88A71.6 71.6 0 0 0 92 143.77a4 4 0 0 0 8 .46a64.3 64.3 0 0 1 2-12.67c0-.12.07-.24.09-.36A64.06 64.06 0 1 1 164 212"/></svg>`,
	CatClear:        `<svg xmlns="http://www.w3.org/2000/svg"  viewBox="0 0 256 256"><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path fill="currentColor" d="M124 40v-8a4 4 0 0 1 8 0v8a4 4 0 0 1-8 0m64 88a60 60 0 1 1-60-60a60.07 60.07 0 0 1 60 60m-8 0a52 52 0 1 0-52 52a52.06 52.06 0 0 0 52-52M61.17 66.83a4 4 0 0 0 5.66-5.66l-8-8a4 4 0 0 0-5.66 5.66Zm0 122.34l-8 8a4 4 0 0 0 5.66 5.66l8-8a4 4 0 0 0-5.66-5.66m136-136l-8 8a4 4 0 0 0 5.66 5.66l8-8a4 4 0 1 0-5.66-5.66m-2.34 136a4 4 0 0 0-5.66 5.66l8 8a4 4 0 0 0 5.66-5.66ZM40 124h-8a4 4 0 0 0 0 8h8a4 4 0 0 0 0-8m88 88a4 4 0 0 0-4 4v8a4 4 0 0 0 8 0v-8a4 4 0 0 0-4-4m96-88h-8a4 4 0 0 0 0 8h8a4 4 0 0 0 0-8"/></svg>`,
	CatCloudy:       `<svg xmlns="http://www.w3.org/2000/svg"  viewBox="0 0 256 256"><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path fill="currentColor" d="M160 44a84.11 84.11 0 0 0-76.41 49.12A60.7 60.7 0 0 0 72 92a60 60 0 0 0 0 120h88a84 84 0 0 0 0-168m0 160H72a52 52 0 1 1 8.55-103.3A83.7 83.7 0 0 0 76 128a4 4 0 0 0 8 0a76 76 0 1 1 76 76"/></svg>`,
	CatStorm:        `<svg xmlns="http://www.w3.org/2000/svg"  viewBox="0 0 256 256"><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path fill="currentColor" d="M156 20a72.19 72.19 0 0 0-68.49 49.39A48 48 0 1 0 76 164h44.94l-20.37 33.94A4 4 0 0 0 104 204h32.94l-20.37 33.94a4 4 0 0 0 6.86 4.12l24-40A4 4 0 0 0 144 196h-32.94l19.2-32H156a72 72 0 0 0 0-144m0 136H76a40 40 0 1 1 9.43-78.88A71.6 71.6 0 0 0 84 87.77a4 4 0 0 0 8 .46A64.06 64.06 0 1 1 156 156"/></svg>`,
	CatRain:         `<svg xmlns="http://www.w3.org/2000/svg"  viewBox="0 0 256 256"><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path fill="currentColor" d="m155.33 194.22l-32 48a4 4 0 1 1-6.66-4.44l32-48a4 4 0 0 1 6.66 4.44M228 92a72.08 72.08 0 0 1-72 72h-25.86l-30.81 46.22a4 4 0 1 1-6.66-4.44L120.53 164H76a48 48 0 1 1 11.51-94.61A72.08 72.08 0 0 1 228 92m-8 0a64.06 64.06 0 0 0-128-3.77a4 4 0 0 1-8-.46a71.6 71.6 0 0 1 1.42-10.65A40 40 0 1 0 76 156h80a64.07 64.07 0 0 0 64-64"/></svg>`,
	CatSnow:         `<svg xmlns="http://www.w3.org/2000/svg"  viewBox="0 0 256 256"><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path fill="currentColor" d="M84 196a8 8 0 1 1-8-8a8 8 0 0 1 8 8m32 8a8 8 0 1 0 8 8a8 8 0 0 0-8-8m48-16a8 8 0 1 0 8 8a8 8 0 0 0-8-8m-96 40a8 8 0 1 0 8 8a8 8 0 0 0-8-8m88 0a8 8 0 1 0 8 8a8 8 0 0 0-8-8m72-136a72.08 72.08 0 0 1-72 72H76a48 48 0 1 1 11.51-94.61A72.08 72.08 0 0 1 228 92m-8 0a64.06 64.06 0 0 0-128-3.77a4 4 0 0 1-8-.46a71.6 71.6 0 0 1 1.42-10.65A40 40 0 1 0 76 156h80a64.07 64.07 0 0 0 64-64"/></svg>`,
	CatSleet:        `<svg xmlns="http://www.w3.org/2000/svg"  viewBox="0 0 256 256"><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path fill="currentColor" d="M84 196a8 8 0 1 1-8-8a8 8 0 0 1 8 8m32 8a8 8 0 1 0 8 8a8 8 0 0 0-8-8m48-16a8 8 0 1 0 8 8a8 8 0 0 0-8-8m-96 40a8 8 0 1 0 8 8a8 8 0 0 0-8-8m88 0a8 8 0 1 0 8 8a8 8 0 0 0-8-8m72-136a72.08 72.08 0 0 1-72 72H76a48 48 0 1 1 11.51-94.61A72.08 72.08 0 0 1 228 92m-8 0a64.06 64.06 0 0 0-128-3.77a4 4 0 0 1-8-.46a71.6 71.6 0 0 1 1.42-10.65A40 40 0 1 0 76 156h80a64.07 64.07 0 0 0 64-64"/></svg>`,
	CatFog:          `<svg xmlns="http://www.w3.org/2000/svg"  viewBox="0 0 256 256"><!-- Icon from Phosphor by Phosphor Icons - https://github.com/phosphor-icons/core/blob/main/LICENSE --><path fill="currentColor" d="M120 204H72a4 4 0 0 1 0-8h48a4 4 0 0 1 0 8m64-8h-24a4 4 0 0 0 0 8h24a4 4 0 0 0 0-8m-24 32h-56a4 4 0 0 0 0 8h56a4 4 0 0 0 0-8m68-128a72.08 72.08 0 0 1-72 72H76a48 48 0 1 1 11.51-94.61A72.08 72.08 0 0 1 228 100m-8 0a64.06 64.06 0 0 0-128-3.77a4 4 0 0 1-8-.46a71.6 71.6 0 0 1 1.42-10.65A40 40 0 1 0 76 164h80a64.07 64.07 0 0 0 64-64"/></svg>`,
}
