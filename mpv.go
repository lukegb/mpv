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

// Package mpv allows authoring Cgo plugins against the MPV plugin API.
package mpv

// #cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-all
// #include <stdlib.h>
// #include <mpv/client.h>
import "C"
import (
	"fmt"
	"log"
	"strconv"
	"unsafe"
)

// Plugin is the minimal possible MPV plugin interface that must be implemented.
// Additional functionality can be enabled by implementing other interfaces, such as RawEventHandler.
type Plugin interface {
	// Open is called when the plugin is initialized.
	// After it returns, the mpv library will run its main event loop.
	Open(*Handle)
}

// RawEventHandler is a Plugin which wishes to receive the raw event data from MPV.
type RawEventHandler interface {
	// HandleEvent is called after mpv's own internal event handling.
	// As a result, it will never receive a shutdown event.
	HandleEvent(*Handle, *C.struct_mpv_event)
}

// Handle represents a handle to the MPV internals, and is opaque.
type Handle struct {
	h *C.struct_mpv_handle

	hooks    map[int]func()
	nextHook int
}

// AddHook registers a new hook function with the given priority.
func (h *Handle) AddHook(hook string, priority int, handler func()) error {
	hookNum := h.nextHook
	h.nextHook++

	h.hooks[hookNum] = handler
	if err := h.Command("hook-add", hook, fmt.Sprintf("%d", hookNum), fmt.Sprintf("%d", priority)); err != nil {
		h.hooks[hookNum] = nil
		return err
	}

	return nil
}

func wrapErr(errCode C.int) error {
	if errCode == 0 {
		return nil
	}
	return fmt.Errorf("mpv: %s", C.GoString(C.mpv_error_string(errCode)))
}

// Command runs an MPV command with the provided string arguments.
func (h *Handle) Command(cmd ...string) error {
	m := make([]*C.char, len(cmd)+1)
	for n, c := range cmd {
		m[n] = C.CString(c)
	}
	m[len(cmd)] = nil

	return wrapErr(C.mpv_command(h.h, &m[0]))
}

// GetPropertyString returns the string value of an MPV property.
func (h *Handle) GetPropertyString(name string) (string, error) {
	var cout *C.char
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	if err := wrapErr(C.mpv_get_property(h.h, cname, C.MPV_FORMAT_STRING, unsafe.Pointer(&cout))); err != nil {
		return "", err
	}

	out := C.GoString(cout)
	C.mpv_free(unsafe.Pointer(cout))
	return out, nil
}

// SetPropertyString sets an MPV property to the provided string value.
func (h *Handle) SetPropertyString(name, data string) error {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	cdata := C.CString(data)
	defer C.free(unsafe.Pointer(cdata))

	return wrapErr(C.mpv_set_property_string(h.h, cname, cdata))
}

var plugin Plugin = nothingPlugin{}

// Register registers your Plugin implementation.
// This should be called inside your main package's init() function.
func Register(p Plugin) {
	plugin = p
}

//export mpv_open_cplugin
func mpv_open_cplugin(handle *C.struct_mpv_handle) {
	h := &Handle{
		h:     handle,
		hooks: make(map[int]func()),
	}
	plugin.Open(h)
	for {
		ev := C.mpv_wait_event(h.h, -1)
		switch ev.event_id {
		case C.MPV_EVENT_SHUTDOWN:
			return

		case C.MPV_EVENT_CLIENT_MESSAGE:
			d := (*C.struct_mpv_event_client_message)(ev.data)
			args := make([]string, d.num_args)
			ah := uintptr(unsafe.Pointer(d.args))
			for n := uintptr(0); n < uintptr(d.num_args); n++ {
				cstr := *(**C.char)(unsafe.Pointer(ah + n*unsafe.Sizeof((*C.char)(nil))))
				args[n] = C.GoString(cstr)
			}

			if len(args) == 3 && args[0] == "hook_run" {
				// handle hook
				n, err := strconv.Atoi(args[1])
				if err != nil {
					log.Printf("mpv: hook message had invalid ID %v", args[1])
					break
				}

				handler, ok := h.hooks[n]
				if !ok {
					log.Printf("mpv: hook message had ID %v, which we have no hook registered for", n)
					break
				}

				handler()

				h.Command("hook-ack", args[2])
			}
		}
		if plugin, ok := plugin.(RawEventHandler); ok {
			plugin.HandleEvent(h, ev)
		}
	}
}

type nothingPlugin struct{}

func (n nothingPlugin) Open(h *Handle) {}
