BASE=github.com/lukegb/mpv

.PHONY: all clean
all: helloworld.so gdrive.so
clean:
	rm helloworld.so || true
	rm gdrive.so || true

.PHONY: install-gdrive
install-gdrive: gdrive.so
	install -d ${HOME}/.config/mpv/scripts
	install $< ${HOME}/.config/mpv/scripts/$< 

helloworld.so: example/helloworld
	go build -buildmode=c-shared -o "$@" "${BASE}/$<"

gdrive.so: example/gdrive
	go build -buildmode=c-shared -o "$@" "${BASE}/$<"
