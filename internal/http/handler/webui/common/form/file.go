package form

import "mime/multipart"

type File struct {
	Header *multipart.FileHeader
	File   multipart.File
}

// hasFileFields checks if any of the dynamic fields are file inputs
func hasFileFields(fields []Field) bool {
	for _, field := range fields {
		if field.IsFile() {
			return true
		}
	}
	return false
}
