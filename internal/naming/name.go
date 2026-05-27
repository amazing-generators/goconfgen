package naming

import "strings"

// // // // // // // // // //

// GoName переводит ключ схемы в идентификатор Go в PascalCase.
// Едина для всех слоёв: schema, ir, codegen.
// // //
// Правила:
//   - подряд идущие латинские буквы и цифры считаются одним словом;
//   - всё прочее — разделитель, после него следующая буква начинает новое слово;
//   - первая буква каждого слова — верхняя; цифра обнуляет регистр;
//   - пустой результат заменяется на "Value", чтобы Go-идентификатор был валидным.
//
// // //
// Не-ASCII символы (кириллица, CJK, эмодзи и т.д.) намеренно
// трактуются как разделители: Go-идентификаторы держим строго ASCII.
// Если ключ "имя_поля" станет коллизионным после трансляции,
// конфликт будет пойман валидатором namespace в ir.Build.
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

// GoTypeName склеивает PascalCase-имена по сегментам пути "a.b.c".
func GoTypeName(pathText string) string {
	partTextArr := strings.Split(pathText, ".")
	var builder strings.Builder

	for _, partText := range partTextArr {
		builder.WriteString(GoName(partText))
	}

	return builder.String()
}
