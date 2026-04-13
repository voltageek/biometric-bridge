// Package tray provides the system tray GUI for the Biometric Bridge.
package tray

import (
	_ "embed"
	// encoding/base64 intentionally removed - not used with generated icon
	"fmt"
	"os/exec"
	"runtime"
	"time"

	cfgpkg "biometric-bridge/internal/config"
	"bytes"
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	systray "fyne.io/systray"
	"golang.design/x/clipboard"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Note: we generate a simple monochrome PNG at runtime for the tray icon.
// This avoids needing external tools or additional build steps.

// StartUI launches a minimal floating panel that displays bridge status and
// provides COPY JWT and RE-SYNC buttons. This implementation focuses on
// User Story 1 (Monitor Bridge Status) and is intentionally small and
// resilient so it can run even when bridge internals (drivers) are not yet
// implemented.
// StartUIWithTray launches the Fyne UI and a system tray icon.
// configPath is used by the tray menu actions (Open Config / Show Log).
func StartUIWithTray(c *BridgeController, jwtStore *JWTStore, evBuf *EventBuffer, logBuf *LogBuffer, configPath string) {
	a := app.New()
	w := a.NewWindow("The Kinetic Vault")
	w.Resize(fyne.NewSize(PanelWidth, 420))

	// Start hidden — the tray is the primary interaction surface.
	w.Hide()

	title := widget.NewLabelWithStyle("The Kinetic Vault", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	statusLabel := widget.NewLabel("status: unknown")
	addrLabel := widget.NewLabel("address: -")
	// Event filter control (type)
	typeOptions := []string{"All", "Scan", "Enrollment", "Connection", "System", "Error"}
	typeSelect := widget.NewSelect(typeOptions, func(string) {})
	typeSelect.SetSelected("All")

	// Event list (simple) - will show recent events
	eventList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {},
	)

	refreshEvents := func() {
		// Determine selected type
		sel := typeSelect.Selected
		var et EventType = EventType(-1)
		switch strings.ToLower(sel) {
		case "scan":
			et = EventTypeScan
		case "enrollment":
			et = EventTypeEnrollment
		case "connection":
			et = EventTypeConnection
		case "system":
			et = EventTypeSystem
		case "error":
			et = EventTypeError
		default:
			et = EventType(-1)
		}

		var items []EventEntry
		if et == EventType(-1) {
			items = evBuf.GetRecent(50)
		} else {
			items = evBuf.GetFiltered(et, Severity(-1))
		}

		// rebuild list backing store by recreating widget (simpler than mutating existing list)
		count := len(items)
		eventList.Length = func() int { return count }
		eventList.CreateItem = func() fyne.CanvasObject { return widget.NewLabel("") }
		eventList.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= count {
				o.(*widget.Label).SetText("")
				return
			}
			e := items[count-1-i] // show newest first
			o.(*widget.Label).SetText(e.Timestamp.Format("15:04:05") + " — " + e.Title + " — " + e.Description)
		}
		eventList.Refresh()
	}

	// hook selection change
	typeSelect.OnChanged = func(string) { refreshEvents() }

	copyBtn := widget.NewButton("COPY JWT", nil)
	resyncBtn := widget.NewButton("RE-SYNC", nil)

	// Initial state
	updateUI := func() {
		st := c.Status()
		statusLabel.SetText("status: " + st.State.String())
		if st.ListenAddr != "" {
			addrLabel.SetText("Running on " + st.ListenAddr)
		} else {
			addrLabel.SetText("Not running")
		}

		// COPY JWT enabled only when token available and valid
		if jwtStore != nil {
			if token, ok := jwtStore.GetToken(); ok && token != "" && jwtStore.IsValid() {
				copyBtn.Enable()
			} else {
				copyBtn.Disable()
			}
		} else {
			copyBtn.Disable()
		}

		// RE-SYNC enabled only when bridge running
		if st.State == StateRunning {
			resyncBtn.Enable()
		} else {
			resyncBtn.Disable()
		}
	}

	// Button actions
	copyBtn.OnTapped = func() {
		if jwtStore == nil {
			return
		}
		if token, ok := jwtStore.GetToken(); ok {
			// Use golang.design/x/clipboard to avoid depending on Fyne clipboard API here
			clipboard.Write(clipboard.MIMEText, []byte(token))
			copyBtn.SetText("COPIED")
			go func() {
				time.Sleep(800 * time.Millisecond)
				copyBtn.SetText("COPY JWT")
			}()
		}
	}

	resyncBtn.OnTapped = func() {
		resyncBtn.Disable()
		go func() {
			if c != nil {
				c.ResyncDevices()
			}
			time.Sleep(700 * time.Millisecond)
			updateUI()
			resyncBtn.Enable()
		}()
	}

	// Layout
	header := container.NewBorder(nil, nil, nil, nil, title)

	body := container.NewVBox(
		container.NewVBox(
			statusLabel,
			addrLabel,
			container.NewHBox(widget.NewLabel("Event Type:"), layout.NewSpacer(), typeSelect),
			eventList,
		),
		layout.NewSpacer(),
		container.NewHBox(copyBtn, layout.NewSpacer(), resyncBtn),
	)

	content := container.NewBorder(header, nil, nil, nil, body)

	// Toast label (hidden by default) shown briefly for operations like Restart/Stop
	toastLabel := widget.NewLabel("")
	toastLabel.Hide()
	toastBox := container.NewVBox(toastLabel)
	// overlay toast on top of content
	overlay := container.NewMax(content, container.NewBorder(toastBox, nil, nil, nil, nil))

	w.SetContent(overlay)

	// showToast displays a short-lived message in the panel. Uses driver.RunOnMain if available.
	showToast := func(msg string, d time.Duration) {
		update := func() {
			toastLabel.SetText(msg)
			toastLabel.Show()
			go func() {
				time.Sleep(d)
				// hide after duration; try to run on main if available
				if drv := a.Driver(); drv != nil {
					if runner, ok := drv.(interface{ RunOnMain(func()) }); ok {
						runner.RunOnMain(func() { toastLabel.Hide() })
						return
					}
				}
				toastLabel.Hide()
			}()
		}
		// try to schedule on main thread if driver supports it
		if drv := a.Driver(); drv != nil {
			if runner, ok := drv.(interface{ RunOnMain(func()) }); ok {
				runner.RunOnMain(update)
				return
			}
		}
		update()
	}

	// Subscribe to status updates
	ch := c.Subscribe()
	go func() {
		for s := range ch {
			// Update UI state based on new status
			statusLabel.SetText("status: " + s.State.String())
			if s.ListenAddr != "" {
				addrLabel.SetText("Running on " + s.ListenAddr)
			} else {
				addrLabel.SetText("Not running")
			}
			updateUI()
		}
	}()

	// Initial UI population
	updateUI()

	// Launch systray in a goroutine — Fyne must run on main goroutine.
	go func() {
		systray.Run(func() {
			// onReady
			// Generate multiple icon sizes and pick an appropriate size.
			sizes := []int{16, 24, 32, 64}
			icons := make(map[int][]byte)
			for _, s := range sizes {
				if b, err := generateDefaultIconPNG(s, s); err == nil {
					icons[s] = b
				}
			}
			// select preferred size
			preferred := 32
			if runtime.GOOS == "darwin" {
				preferred = 64
			} else if runtime.GOOS == "windows" {
				preferred = 32
			}
			if env := strings.TrimSpace(os.Getenv("TRAY_ICON_SIZE")); env != "" {
				if vs, err := strconv.Atoi(env); err == nil {
					preferred = vs
				}
			}
			iconData := icons[preferred]
			if len(iconData) == 0 {
				// fallback to any available size
				for _, s := range sizes {
					if b, ok := icons[s]; ok {
						iconData = b
						break
					}
				}
			}
			if len(iconData) > 0 {
				systray.SetIcon(iconData)
			}
			systray.SetTitle("Kinetic Vault")
			systray.SetTooltip("The Kinetic Vault — Biometric Bridge")

			mShow := systray.AddMenuItem("Show Panel", "Show the Kinetic Vault panel")
			mOpenCfg := systray.AddMenuItem("Open Config File", "Open bridge config in editor")
			mShowLog := systray.AddMenuItem("Show Log Folder", "Open folder containing bridge log file")
			mCheckUpdates := systray.AddMenuItem("Check for Updates", "Check for new application versions (MVP: stub)")
			systray.AddSeparator()
			mRestart := systray.AddMenuItem("Restart Service", "Restart the embedded bridge")
			mStop := systray.AddMenuItem("Stop Service", "Stop the embedded bridge")
			systray.AddSeparator()
			mQuit := systray.AddMenuItem("Quit", "Quit The Kinetic Vault")

			// Handle clicks
			// track visibility locally to avoid depending on Window.Visible API
			windowVisible := false
			go func() {
				for {
					select {
					case <-mShow.ClickedCh:
						// Toggle visibility
						if windowVisible {
							w.Hide()
							windowVisible = false
						} else {
							// Try to position the window at top-right of primary screen using a best-effort detector.
							sw, _ := detectScreenSize()
							x := sw - PanelWidth - 20
							y := 40
							// Attempt to set position using SetPosition if available (best-effort).
							if mover, ok := w.(interface{ SetPosition(fyne.Position) }); ok {
								mover.SetPosition(fyne.NewPos(float32(x), float32(y)))
							} else {
								// No SetPosition available; nothing to do. Fyne will choose placement.
							}

							// Show the window
							w.Show()
							w.RequestFocus()
							windowVisible = true
						}
					case <-mOpenCfg.ClickedCh:
						// Open config file in default editor/viewer
						go openFile(configPath)
					case <-mShowLog.ClickedCh:
						// Determine log location from configPath — safely try to load config and open its log file folder.
						go func() {
							// default to current dir
							folder := "./"
							if strings.TrimSpace(configPath) != "" {
								if cfg, err := cfgpkg.LoadDriverOnly(configPath); err == nil {
									if cfg.Log.File != "" {
										folder = filepath.Dir(cfg.Log.File)
									}
								}
							}
							_ = openFolder(folder)
						}()
					case <-mCheckUpdates.ClickedCh:
						go func() {
							// Determine configured update URL from config if available
							var updateURL string
							var ver string
							if strings.TrimSpace(configPath) != "" {
								if cfg, err := cfgpkg.LoadDriverOnly(configPath); err == nil {
									updateURL = cfg.Tray.UpdateCheckURL
									ver = cfg.Tray.Version
								}
							}
							if ver == "" {
								ver = Version
							}
							res := CheckForUpdates(ver, updateURL)
							showToast(res, 4*time.Second)
						}()
					case <-mQuit.ClickedCh:
						systray.Quit()
						// Ensure Fyne app exits as well
						a.Quit()
						return
					case <-mRestart.ClickedCh:
						go func() {
							// best-effort restart with timeout
							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()
							if err := c.Restart(ctx); err != nil {
								showToast("Restart failed: "+err.Error(), 3*time.Second)
							} else {
								showToast("Restart successful", 2*time.Second)
							}
						}()
					case <-mStop.ClickedCh:
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
							defer cancel()
							if err := c.Stop(ctx); err != nil {
								showToast("Stop failed: "+err.Error(), 3*time.Second)
							} else {
								showToast("Stopped", 2*time.Second)
							}
						}()
					}
				}
			}()

		}, func() {
			// onExit
		})
	}()

	// Ensure the close button hides the window instead of exiting the app.
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	// Run Fyne (blocks on main goroutine)
	a.Run()
}

// openFile opens a file with the platform default application.
func openFile(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("cmd", "/C", "start", "", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

// openFolder opens a folder with the platform default file manager.
func openFolder(path string) error {
	return openFile(path)
}

// generateDefaultIconPNG creates a small PNG image (navy background with a
// white inset square) and returns its bytes. This is a minimal programmatic
// icon for development and defaults.
func generateDefaultIconPNG(w, h int) ([]byte, error) {
	// Create an RGBA image
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	navy := color.RGBA{15, 22, 35, 255} // #0F1623
	white := color.RGBA{255, 255, 255, 255}
	// fill background
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, navy)
		}
	}
	// draw inset white square
	inset := w / 6
	for y := inset; y < h-inset; y++ {
		for x := inset; x < w-inset; x++ {
			img.Set(x, y, white)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// detectScreenSize attempts to determine primary screen width and height.
// It uses simple platform-specific commands as a best-effort and falls back
// to a sensible default when detection fails.
func detectScreenSize() (width, height int) {
	// sensible default
	width, height = 1440, 900
	switch runtime.GOOS {
	case "linux":
		// try xrandr to get primary resolution
		out, err := exec.Command("bash", "-lc", "xrandr | grep '*' | awk '{print $1; exit}'").Output()
		if err == nil {
			s := strings.TrimSpace(string(out))
			if s != "" {
				var w, h int
				if _, err := fmt.Sscanf(s, "%dx%d", &w, &h); err == nil {
					return w, h
				}
			}
		}
	case "darwin":
		out, err := exec.Command("bash", "-lc", "system_profiler SPDisplaysDataType | awk '/Resolution/{print $2 \"x\" $4; exit}'").Output()
		if err == nil {
			s := strings.TrimSpace(string(out))
			var w, h int
			if _, err := fmt.Sscanf(s, "%dx%d", &w, &h); err == nil {
				return w, h
			}
		}
	case "windows":
		// leave default for now
	}
	return width, height
}
