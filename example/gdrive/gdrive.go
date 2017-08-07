/*
Copyright 2017 Luke Granger-Brown

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"log"
	"os/user"
	"path/filepath"
	"regexp"

	"github.com/lukegb/mpv"
)

var (
	gdriveRe = regexp.MustCompile(`^https:\/\/drive.google.com\/open\?(?:[^&]*&)*id=([^&]*)(?:&|$)`)
)

type Plugin struct{}

func (Plugin) Open(h *mpv.Handle) {
	h.AddHook("on_load", 5, func() {
		url, err := h.GetPropertyString("stream-open-filename")
		if err != nil {
			log.Printf("gdrive: failed to get property: %v", err)
			return
		}

		// do we care about this URL?
		m := gdriveRe.FindStringSubmatch(url)
		if m == nil {
			// we do not
			return
		}

		driveID := m[1]
		log.Println("Authenticating request for " + driveID + " against Google Drive")

		newURL := "https://www.googleapis.com/drive/v3/files/" + driveID + "?alt=media"

		usr, err := user.Current()
		if err != nil {
			log.Printf("failed to get current user: %v", err)
			return
		}

		tok, err := fetchToken(context.Background(), filepath.Join(usr.HomeDir, ".config", "mpv-gdrive-oauth-token.json"))
		if err != nil {
			log.Printf("failed to fetch OAuth2 token: %v", err)
			return
		}

		h.SetPropertyString("stream-open-filename", newURL)
		h.SetPropertyString("file-local-options/http-header-fields", "Authorization: Bearer "+tok)
	})
}

func init() {
	var p Plugin
	mpv.Register(p)
}

func main() {}
