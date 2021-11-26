package signature

import "strings"

// StringPosToLineNumber convert pos index to line number of the content
func StringPosToLineNumber(s string, pos int) int32 {
	if pos < 0 {
		return int32(-1)
	}

	lines := strings.Split(s, "\n")
	curPos := 0

	for lineNumber, line := range lines {
		curLine := len(line)
		if curPos+curLine > pos {
			return int32(lineNumber)
		}
		curPos += curLine + 1
	}

	return int32(-1)
}
