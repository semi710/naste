package handlers

import "fmt"

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

func highlightSyntax(code string) string {
	keywords := map[string]bool{
		"func": true, "var": true, "const": true, "type": true, "struct": true,
		"interface": true, "package": true, "import": true, "return": true,
		"if": true, "else": true, "for": true, "range": true, "switch": true,
		"case": true, "default": true, "break": true, "continue": true,
		"map": true, "chan": true, "go": true, "defer": true, "select": true,
		"true": true, "false": true, "nil": true, "int": true, "string": true,
		"bool": true, "float64": true, "error": true, "make": true, "new": true,
		"append": true, "len": true, "cap": true, "copy": true, "delete": true,
		"fmt": true, "log": true, "http": true, "os": true, "io": true,
	}

	var result []byte
	i := 0
	for i < len(code) {
		if i+1 < len(code) && code[i] == '/' && code[i+1] == '/' {
			end := i + 2
			for end < len(code) && code[end] != '\n' {
				end++
			}
			result = append(result, `<span class="comment">`...)
			for j := i; j < end; j++ {
				result = append(result, htmlEscape(code[j])...)
			}
			result = append(result, "</span>"...)
			i = end
			continue
		}

		if code[i] == '"' || code[i] == '`' {
			quote := code[i]
			end := i + 1
			for end < len(code) && code[end] != quote {
				if code[end] == '\\' && end+1 < len(code) {
					end += 2
				} else {
					end++
				}
			}
			if end < len(code) {
				end++
			}
			result = append(result, `<span class="string">`...)
			for j := i; j < end; j++ {
				result = append(result, htmlEscape(code[j])...)
			}
			result = append(result, "</span>"...)
			i = end
			continue
		}

		if isDigit(code[i]) {
			start := i
			for i < len(code) && (isDigit(code[i]) || code[i] == '.' || code[i] == '_') {
				i++
			}
			result = append(result, `<span class="number">`...)
			result = append(result, code[start:i]...)
			result = append(result, "</span>"...)
			continue
		}

		if isWordChar(code[i]) {
			start := i
			for i < len(code) && isWordChar(code[i]) {
				i++
			}
			word := code[start:i]
			if keywords[word] {
				result = append(result, fmt.Sprintf(`<span class="keyword">%s</span>`, escapeHTML(word))...)
			} else if i < len(code) && code[i] == '(' {
				result = append(result, fmt.Sprintf(`<span class="function">%s</span>`, escapeHTML(word))...)
			} else {
				result = append(result, escapeHTML(word)...)
			}
			continue
		}

		if isOperator(code[i]) {
			result = append(result, `<span class="operator">`...)
			result = append(result, escapeHTML(string(code[i]))...)
			result = append(result, "</span>"...)
			i++
			continue
		}

		result = append(result, htmlEscape(code[i])...)
		i++
	}

	return string(result)
}

func isDigit(c byte) bool    { return c >= '0' && c <= '9' }
func isWordChar(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' }
func isOperator(c byte) bool {
	return c == '+' || c == '-' || c == '*' || c == '/' || c == '=' || c == '<' || c == '>' ||
		c == '!' || c == '&' || c == '|' || c == '^' || c == '~' || c == '%'
}

func htmlEscape(c byte) []byte {
	switch c {
	case '<':
		return []byte("&lt;")
	case '>':
		return []byte("&gt;")
	case '&':
		return []byte("&amp;")
	case '\t':
		return []byte("    ")
	default:
		return []byte{c}
	}
}

func escapeHTML(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		result = append(result, htmlEscape(s[i])...)
	}
	return string(result)
}
