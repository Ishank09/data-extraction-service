package utils

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
)

// BytesToJSON converts bytes to structured JSON format (auto-detects format)
func BytesToJSON(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return map[string]interface{}{
			"content": "",
			"type":    "empty",
		}, nil
	}

	content := string(data)

	// Auto-detect format and convert accordingly
	if isHTML(content) {
		return OneNoteHTMLToJSON(content)
	}

	if isXML(content) {
		return XMLToJSON(content)
	}

	if isMarkdown(content) {
		return MarkdownToJSON(content)
	}

	if isJSON(content) {
		return parseExistingJSON(content)
	}

	// Default to plain text
	return TextToJSON(content)
}

// OneNoteHTMLToJSON converts OneNote HTML content to clean JSON format
func OneNoteHTMLToJSON(htmlContent string) (map[string]interface{}, error) {
	if htmlContent == "" {
		return map[string]interface{}{
			"content": "",
			"type":    "empty",
		}, nil
	}

	// Extract text content from HTML
	textContent := HTMLToText(htmlContent)

	// Extract structured data from OneNote HTML
	sections := extractOneNoteSections(htmlContent)

	result := map[string]interface{}{
		"content":         textContent,
		"type":            "onenote_html",
		"sections":        sections,
		"has_images":      hasImages(htmlContent),
		"has_tables":      hasTables(htmlContent),
		"has_links":       hasLinks(htmlContent),
		"word_count":      countWords(textContent),
		"character_count": len(textContent),
	}

	return result, nil
}

// TextToJSON converts plain text to structured JSON format
func TextToJSON(textContent string) (map[string]interface{}, error) {
	if textContent == "" {
		return map[string]interface{}{
			"content": "",
			"type":    "empty",
		}, nil
	}

	// Extract structured data from text
	paragraphs := extractParagraphs(textContent)
	headers := extractHeaders(textContent)

	result := map[string]interface{}{
		"content":         textContent,
		"type":            "text",
		"paragraphs":      paragraphs,
		"headers":         headers,
		"word_count":      countWords(textContent),
		"character_count": len(textContent),
	}

	return result, nil
}

// XMLToJSON converts XML content to structured JSON format
func XMLToJSON(xmlContent string) (map[string]interface{}, error) {
	if xmlContent == "" {
		return map[string]interface{}{
			"content": "",
			"type":    "empty",
		}, nil
	}

	// Extract text content from XML
	textContent := XMLToText(xmlContent)

	// Extract structured data from XML
	elements := extractXMLElements(xmlContent)

	result := map[string]interface{}{
		"content":         textContent,
		"type":            "xml",
		"elements":        elements,
		"word_count":      countWords(textContent),
		"character_count": len(textContent),
	}

	return result, nil
}

// MarkdownToJSON converts Markdown content to structured JSON format
func MarkdownToJSON(markdownContent string) (map[string]interface{}, error) {
	if markdownContent == "" {
		return map[string]interface{}{
			"content": "",
			"type":    "empty",
		}, nil
	}

	// Extract text content from Markdown
	textContent := MarkdownToText(markdownContent)

	// Extract structured data from Markdown
	headers := extractMarkdownHeaders(markdownContent)
	links := extractMarkdownLinks(markdownContent)

	result := map[string]interface{}{
		"content":         textContent,
		"type":            "markdown",
		"headers":         headers,
		"links":           links,
		"word_count":      countWords(textContent),
		"character_count": len(textContent),
	}

	return result, nil
}

// HTMLToText extracts clean text from HTML content
func HTMLToText(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	// Remove HTML tags
	htmlTagRe := regexp.MustCompile(`<[^>]*>`)
	plainText := htmlTagRe.ReplaceAllString(htmlContent, "")

	// Decode HTML entities
	plainText = html.UnescapeString(plainText)

	// Clean up whitespace
	plainText = strings.TrimSpace(plainText)
	plainText = regexp.MustCompile(`\s+`).ReplaceAllString(plainText, " ")

	return plainText
}

// XMLToText extracts clean text from XML content
func XMLToText(xmlContent string) string {
	if xmlContent == "" {
		return ""
	}

	// Remove XML tags
	xmlTagRe := regexp.MustCompile(`<[^>]*>`)
	plainText := xmlTagRe.ReplaceAllString(xmlContent, " ")

	// Clean up whitespace
	plainText = strings.TrimSpace(plainText)
	plainText = regexp.MustCompile(`\s+`).ReplaceAllString(plainText, " ")

	return plainText
}

// MarkdownToText extracts clean text from Markdown content
func MarkdownToText(markdownContent string) string {
	if markdownContent == "" {
		return ""
	}

	text := markdownContent

	// Remove markdown formatting
	// Links: [text](url) -> text
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	text = linkRe.ReplaceAllString(text, "$1")

	// Headers: # Header -> Header
	headerRe := regexp.MustCompile(`#+\s*(.+)`)
	text = headerRe.ReplaceAllString(text, "$1")

	// Bold/Italic: **text** or *text* -> text
	boldRe := regexp.MustCompile(`\*+([^*]+)\*+`)
	text = boldRe.ReplaceAllString(text, "$1")

	// Code blocks: ```code``` -> code
	codeBlockRe := regexp.MustCompile("```[^`]*```")
	text = codeBlockRe.ReplaceAllString(text, "")

	// Inline code: `code` -> code
	inlineCodeRe := regexp.MustCompile("`([^`]+)`")
	text = inlineCodeRe.ReplaceAllString(text, "$1")

	// Clean up whitespace
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text
}

// Detection functions
func isHTML(content string) bool {
	return regexp.MustCompile(`<[^>]+>`).MatchString(content)
}

func isXML(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "<?xml") || regexp.MustCompile(`<[^>]+>`).MatchString(content)
}

func isMarkdown(content string) bool {
	return regexp.MustCompile(`(^|\n)#+\s|^\*+\s|\[.+\]\(.+\)|` + "`").MatchString(content)
}

func isJSON(content string) bool {
	content = strings.TrimSpace(content)
	return (strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}")) ||
		(strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]"))
}

// Extraction functions
func extractOneNoteSections(htmlContent string) []map[string]interface{} {
	sections := []map[string]interface{}{}

	// Extract sections based on common OneNote patterns
	sectionRe := regexp.MustCompile(`<div[^>]*>(.*?)</div>`)
	matches := sectionRe.FindAllStringSubmatch(htmlContent, -1)

	for i, match := range matches {
		if len(match) > 1 {
			sectionText := HTMLToText(match[1])
			if strings.TrimSpace(sectionText) != "" {
				sections = append(sections, map[string]interface{}{
					"index":   i,
					"content": sectionText,
				})
			}
		}
	}

	return sections
}

func extractParagraphs(textContent string) []string {
	paragraphs := []string{}

	// Split by double newlines or single newlines for paragraph detection
	lines := strings.Split(textContent, "\n")
	currentParagraph := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentParagraph != "" {
				paragraphs = append(paragraphs, currentParagraph)
				currentParagraph = ""
			}
		} else {
			if currentParagraph != "" {
				currentParagraph += " "
			}
			currentParagraph += line
		}
	}

	if currentParagraph != "" {
		paragraphs = append(paragraphs, currentParagraph)
	}

	return paragraphs
}

func extractHeaders(textContent string) []map[string]interface{} {
	headers := []map[string]interface{}{}

	lines := strings.Split(textContent, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		// Simple heuristic: lines that are short and followed by empty line or end
		if len(line) > 0 && len(line) < 100 {
			if i == len(lines)-1 || strings.TrimSpace(lines[i+1]) == "" {
				headers = append(headers, map[string]interface{}{
					"line":    i + 1,
					"content": line,
				})
			}
		}
	}

	return headers
}

func extractXMLElements(xmlContent string) []string {
	elements := []string{}

	// Extract XML element names
	elementRe := regexp.MustCompile(`<([^>/\s]+)`)
	matches := elementRe.FindAllStringSubmatch(xmlContent, -1)

	seenElements := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			element := match[1]
			if !seenElements[element] {
				elements = append(elements, element)
				seenElements[element] = true
			}
		}
	}

	return elements
}

func extractMarkdownHeaders(markdownContent string) []map[string]interface{} {
	headers := []map[string]interface{}{}

	lines := strings.Split(markdownContent, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			// Count header level
			level := 0
			for _, char := range line {
				if char == '#' {
					level++
				} else {
					break
				}
			}

			headerText := strings.TrimSpace(strings.TrimPrefix(line, strings.Repeat("#", level)))
			headers = append(headers, map[string]interface{}{
				"line":    i + 1,
				"level":   level,
				"content": headerText,
			})
		}
	}

	return headers
}

func extractMarkdownLinks(markdownContent string) []map[string]interface{} {
	links := []map[string]interface{}{}

	// Extract markdown links: [text](url)
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := linkRe.FindAllStringSubmatch(markdownContent, -1)

	for _, match := range matches {
		if len(match) > 2 {
			links = append(links, map[string]interface{}{
				"text": match[1],
				"url":  match[2],
			})
		}
	}

	return links
}

func hasImages(htmlContent string) bool {
	return regexp.MustCompile(`<img[^>]*>`).MatchString(htmlContent)
}

func hasTables(htmlContent string) bool {
	return regexp.MustCompile(`<table[^>]*>`).MatchString(htmlContent)
}

func hasLinks(htmlContent string) bool {
	return regexp.MustCompile(`<a[^>]*>`).MatchString(htmlContent)
}

func countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func parseExistingJSON(content string) (map[string]interface{}, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"content":         content,
		"type":            "json",
		"parsed_data":     data,
		"character_count": len(content),
	}

	return result, nil
}
