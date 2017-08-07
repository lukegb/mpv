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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	Scopes = []string{"https://www.googleapis.com/auth/drive.readonly", "email", "profile"}

	OAuthConfig = &oauth2.Config{
		ClientID:     "1032160245384-6pboet8pqv0p409iic5kugvq54v0egtf.apps.googleusercontent.com",
		ClientSecret: "gw7azXzMc26FeNuQGnoFRM0R",
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		Scopes:       Scopes,
		Endpoint:     google.Endpoint,
	}
)

type tokenServer struct {
	srv http.Server
	tok chan string
}

func (ts *tokenServer) Start() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, nil
	}

	go ts.srv.Serve(l)

	return l.Addr().(*net.TCPAddr).Port, nil
}

func (ts *tokenServer) Stop() error {
	return ts.srv.Close()
}

func (ts *tokenServer) GetToken() string {
	return <-ts.tok
}

func newTokenServer() *tokenServer {
	sm := http.NewServeMux()
	ts := &tokenServer{
		tok: make(chan string),
	}
	sm.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code, ok := r.URL.Query()["code"]
		if !ok {
			http.Error(w, "no 'code' query parameter", http.StatusBadRequest)
			return
		}
		if len(code) != 1 {
			http.Error(w, "'code' specified multiple times", http.StatusBadRequest)
			return
		}

		ts.tok <- code[0]
		fmt.Fprintf(w, "Thanks! Authorization complete. You can now close this tab.")
	})

	ts.srv.Handler = sm
	return ts
}

func loadToken(tokenPath string) (*oauth2.Token, error) {
	f, err := os.Open(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var t oauth2.Token
	if err := json.NewDecoder(f).Decode(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

func saveToken(tokenPath string, t *oauth2.Token) error {
	fn := tokenPath + ".new"
	f, err := os.Create(fn)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(f).Encode(t); err != nil {
		f.Close()
		os.Remove(fn)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(fn)
		return err
	}

	if err := os.Rename(fn, tokenPath); err != nil {
		os.Remove(fn)
		return err
	}
	return nil
}

func fetchToken(ctx context.Context, tokenPath string) (string, error) {
	tok, err := loadToken(tokenPath)
	if err != nil {
		return "", err
	}

	if tok == nil {
		ts := newTokenServer()
		tsPort, err := ts.Start()
		defer ts.Stop()
		if err != nil {
			return "", err
		}

		OAuthConfig.RedirectURL = fmt.Sprintf("http://localhost:%v", tsPort)
		url := OAuthConfig.AuthCodeURL("", oauth2.AccessTypeOffline)
		fmt.Printf("\n\n\n\n\nVisit the URL for the auth dialog: %v\n", url)

		code := ts.GetToken()
		fmt.Println("Token retrieved! Please wait a moment...")
		tok, err = OAuthConfig.Exchange(ctx, code)
		if err != nil {
			return "", err
		}

		if err := saveToken(tokenPath, tok); err != nil {
			return "", err
		}
	}

	if !tok.Valid() {
		src := OAuthConfig.TokenSource(ctx, tok)

		var err error
		tok, err = src.Token()
		if err != nil {
			return "", err
		}

		if err := saveToken(tokenPath, tok); err != nil {
			return "", err
		}
	}

	return tok.AccessToken, nil
}
