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

package apikey

import (
	"net/http"

	"github.com/google/exposure-notifications-verification-server/pkg/controller"
	"github.com/google/exposure-notifications-verification-server/pkg/database"
)

func (c *Controller) HandleIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		realm := controller.RealmFromContext(ctx)
		if realm == nil {
			controller.MissingRealm(w, r, c.h)
			return
		}

		// Perform the lazy load on authorized apps for the realm.
		if _, err := realm.GetAuthorizedApps(c.db, true); err != nil {
			controller.InternalError(w, r, c.h, err)
			return
		}

		c.renderIndex(w, r, realm)
	})
}

// renderIndex renders the index page.
func (c *Controller) renderIndex(w http.ResponseWriter, r *http.Request, realm *database.Realm) {
	ctx := r.Context()
	m := controller.TemplateMapFromContext(ctx)
	m["apps"] = realm.AuthorizedApps
	c.h.RenderHTML(w, "apikeys/index", m)
}
