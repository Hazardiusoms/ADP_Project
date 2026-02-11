package auth

import (
	"log"
	"net/http"
)

// SafeParseForm safely parses form data, handling both regular and multipart forms
func SafeParseForm(r *http.Request) error {
	// Check if form is already parsed
	if r.Form != nil || r.MultipartForm != nil {
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	
	// Try multipart first if Content-Type suggests it
	if contentType != "" && len(contentType) > 19 && contentType[:19] == "multipart/form-data" {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			log.Printf("Error parsing multipart form: %v, Content-Type: %s", err, contentType)
			return err
		}
		return nil
	}

	// Try regular form parsing
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v, Content-Type: %s, Method: %s", err, contentType, r.Method)
		return err
	}

	return nil
}
