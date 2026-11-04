package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// The Blender bridge: a server on localhost the add-on talks to. The add-on finds
// its port and token in bridge.json in the config dir; only callers with the token
// get through.
type bridge struct {
	mu     sync.Mutex
	token  string
	file   string
	blends map[string]*blendState
}

// blendState is what texxer has for one .blend: the texture's files and a count
// that goes up with each new one, and a count of syncs texxer asked for.
type blendState struct {
	Texture  int    `json:"texture"`
	Color    string `json:"color"`
	Emissive string `json:"emissive"`
	// Glow is the emission strength, as the export writes it.
	Glow float64 `json:"glow"`
	Sync int     `json:"sync"`
	seen time.Time
}

// BridgeSync is what the add-on sends after writing a .blend's objects into a GLB.
type BridgeSync struct {
	Blend string `json:"blend"`
	GLB   string `json:"glb"`
	Name  string `json:"name"`
}

func bridgeFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "texxer", "bridge.json"), nil
}

func (a *App) startBridge() error {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	file, err := bridgeFile()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	b := &bridge{token: hex.EncodeToString(key), file: file, blends: map[string]*blendState{}}
	a.bridge = b

	info, _ := json.Marshal(map[string]any{"port": ln.Addr().(*net.TCPAddr).Port, "token": b.token})
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(file, info, 0o600); err != nil {
		return err
	}
	go http.Serve(ln, a.bridgeHandler())
	return nil
}

func (a *App) stopBridge() {
	if a.bridge != nil {
		os.Remove(a.bridge.file)
	}
}

func (b *bridge) state(blend string) *blendState {
	s, ok := b.blends[blend]
	if !ok {
		s = &blendState{}
		b.blends[blend] = s
	}
	return s
}

func (a *App) bridgeHandler() http.Handler {
	b := a.bridge
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Texxer-Token")), []byte(b.token)) != 1 {
			http.Error(w, "bad token", http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/state":
			blend := r.URL.Query().Get("blend")
			b.mu.Lock()
			s := b.state(blend)
			s.seen = time.Now()
			out, _ := json.Marshal(s)
			b.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.Write(out)

		case "/sync":
			var s BridgeSync
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if !filepath.IsAbs(s.Blend) || !strings.EqualFold(filepath.Ext(s.GLB), ".glb") || !exists(s.GLB) {
				http.Error(w, "bad sync", http.StatusBadRequest)
				return
			}
			runtime.EventsEmit(a.ctx, "bridge:sync", s)
			runtime.WindowShow(a.ctx)

		default:
			http.NotFound(w, r)
		}
	})
}

// BridgeConnected lists the .blend files whose add-on asked in the last seconds.
func (a *App) BridgeConnected() []string {
	out := []string{}
	if a.bridge == nil {
		return out
	}
	b := a.bridge
	b.mu.Lock()
	defer b.mu.Unlock()
	for blend, s := range b.blends {
		if time.Since(s.seen) < 3*time.Second {
			out = append(out, blend)
		}
	}
	return out
}

// BridgeTexture writes a model's live texture where its export puts it, for the
// .blend to show, and tells the add-on.
func (a *App) BridgeTexture(blend, dir, name, color, emissive string, glow float64) error {
	if a.bridge == nil {
		return errors.New("no bridge")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := filepath.Join(dir, fileName(name))
	if err := writePNGs(base, color, emissive); err != nil {
		return err
	}
	b := a.bridge
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.state(blend)
	s.Texture++
	s.Color = base + ".png"
	s.Glow = glow
	s.Emissive = ""
	if emissive != "" {
		s.Emissive = base + "_emissive.png"
	}
	return nil
}

// BridgeRequestSync asks the add-on of a .blend to send its objects again.
func (a *App) BridgeRequestSync(blend string) {
	if a.bridge == nil {
		return
	}
	a.bridge.mu.Lock()
	defer a.bridge.mu.Unlock()
	a.bridge.state(blend).Sync++
}

// PickExportTarget asks where a bridge model's export goes.
func (a *App) PickExportTarget(name string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export to",
		DefaultFilename: fileName(name) + ".glb",
		Filters:         []runtime.FileFilter{{DisplayName: "glTF binary", Pattern: "*.glb"}},
	})
}
