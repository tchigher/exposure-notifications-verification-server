// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RenderJSON renders the interface as JSON. It attempts to gracefully handle
// any rendering errors to avoid partial responses sent to the response by
// writing to a buffer first, then flushing the buffer to the response.
//
// If the provided data is nil and the response code is a 200, the result will
// be `{"ok":true}`. If the code is not a 200, the response will be of the
// format `{"error":"<val>"}` where val is the JSON-escaped http.StatusText for
// the provided code.
//
// If rendering fails, a generic 500 JSON response is returned. In dev mode, the
// error is included in the payload. If flushing the buffer to the response
// fails, an error is logged, but no recovery is attempted.
//
// The buffers are fetched via a sync.Pool to reduce allocations and improve
// performance.
func (r *Renderer) RenderJSON(w http.ResponseWriter, code int, data interface{}) {
	// Avoid marshaling nil data.
	if data == nil {
		w.WriteHeader(code)
		w.Header().Set("Content-Type", "application/json")

		// Return an OK response.
		if code >= 200 && code < 300 {
			fmt.Fprint(w, jsonOKResp)
			return
		}

		// Return an error with the generic HTTP text as the error.
		msg := escapeJSON(http.StatusText(code))
		fmt.Fprintf(w, jsonErrTmpl, msg)
		return
	}

	// Acquire a renderer
	b := r.rendererPool.Get().(*bytes.Buffer)
	b.Reset()
	defer r.rendererPool.Put(b)

	// Render into the renderer
	if err := json.NewEncoder(b).Encode(data); err != nil {
		msg := "An internal error occurred."
		if r.debug {
			msg = err.Error()
		}
		msg = escapeJSON(msg)

		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, jsonErrTmpl, msg)
		return
	}

	// Rendering worked, flush to the response
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	if _, err := b.WriteTo(w); err != nil {
		// We couldn't write the buffer. We can't change the response header or
		// content type if we got this far, so the best option we have is to log the
		// error.
		r.logger.Errorw("failed to write json to response", "error", err)
	}
}

// JSON500 renders the given error as JSON. In production mode, this always
// renders a generic "server error" message. In debug, it returns the actual
// error from the caller.
func (r *Renderer) JSON500(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	if r.debug {
		r.RenderJSON(w, code, map[string]string{
			"error": err.Error(),
		})
		return
	}

	r.RenderJSON(w, code, map[string]string{
		"error": http.StatusText(code),
	})
}

// escapeJSON does primitive JSON escaping.
func escapeJSON(s string) string {
	return strings.Replace(s, `"`, `\"`, -1)
}

// jsonErrTmpl is the template to use when returning a JSON error. It is
// rendered using Printf, not json.Encode, so values must be escaped by the
// caller.
const jsonErrTmpl = `{"error":"%s"}`

// jsonOKResp is the return value for empty data responses.
const jsonOKResp = `{"ok":true}`
