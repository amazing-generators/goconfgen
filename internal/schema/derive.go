package schema

import "strings"

// // // // // // // // // //

// GoName переводит ключ схемы в PascalCase-идентификатор.
func GoName(textValue string) string {
	var builder strings.Builder
	upperFlag := true

	for _, itemRune := range textValue {
		switch {
		case itemRune >= 'a' && itemRune <= 'z':
			if upperFlag {
				builder.WriteRune(itemRune - 32)
			} else {
				builder.WriteRune(itemRune)
			}
			upperFlag = false
		case itemRune >= 'A' && itemRune <= 'Z':
			if upperFlag {
				builder.WriteRune(itemRune)
			} else {
				builder.WriteRune(itemRune + 32)
			}
			upperFlag = false
		case itemRune >= '0' && itemRune <= '9':
			builder.WriteRune(itemRune)
			upperFlag = true
		default:
			upperFlag = true
		}
	}

	if builder.Len() == 0 {
		return "Value"
	}

	return builder.String()
}

func GoTypeName(pathText string) string {
	partTextArr := strings.Split(pathText, ".")
	var builder strings.Builder

	for _, partText := range partTextArr {
		builder.WriteString(GoName(partText))
	}

	return builder.String()
}
