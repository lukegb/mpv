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
	"log"

	"github.com/lukegb/mpv"
)

type Plugin struct{}

func (Plugin) Open(h *mpv.Handle) {
	h.AddHook("on_load", 5, func() {
		fn, err := h.GetPropertyString("stream-open-filename")
		if err != nil {
			panic(err)
		}

		log.Println("Hello world, from on_load, playing", fn)

		if err := h.SetPropertyString("stream-open-filename", "https://www.youtube.com/watch?v=dQw4w9WgXcQ"); err != nil {
			panic(err)
		}
	})
}

func init() {
	var p Plugin
	mpv.Register(p)
}

func main() {}
