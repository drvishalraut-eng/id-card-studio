// Command idcard runs the ID Card Studio server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"idcardstudio/internal/activity"
	"idcardstudio/internal/auth"
	"idcardstudio/internal/clients"
	"idcardstudio/internal/config"
	"idcardstudio/internal/employees"
	"idcardstudio/internal/export"
	"idcardstudio/internal/netutil"
	"idcardstudio/internal/presence"
	"idcardstudio/internal/server"
	"idcardstudio/web"
)

func main() {
	appDir, err := config.AppDir()
	if err != nil {
		log.Fatalf("find application directory: %v", err)
	}
	cfg, err := config.Load(appDir)
	if err != nil {
		log.Fatalf("load config.json: %v", err)
	}
	data, err := config.EnsureDataDir(appDir)
	if err != nil {
		log.Fatalf("prepare data folder: %v", err)
	}

	authManager := auth.NewManager(filepath.Join(data.Root, "users.json"))
	activityLog := activity.New(filepath.Join(data.Root, "activity.jsonl"))
	presenceManager := presence.New()
	clientsManager := clients.NewManager(filepath.Join(data.Root, "clients.json"))
	if err := clientsManager.SeedHelios(); err != nil {
		log.Fatalf("seed Helios client: %v", err)
	}
	employeesManager := employees.NewManager(filepath.Join(data.Root, "employees.json"))
	exportManager := export.New(employeesManager, cfg.ExportDir)
	info := server.Info{
		StartedAt: time.Now(),
		Port:      cfg.Port,
		DataDir:   data.Root,
		ExportDir: cfg.ExportDir,
		PhotosDir: data.Photos,
	}
	api := server.New(authManager, activityLog, presenceManager, clientsManager, employeesManager, exportManager, info)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", api.Handler()))
	mux.Handle("/", http.FileServer(http.FS(web.FS)))

	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", cfg.Port),
		Handler: mux,
	}

	urls := netutil.LANURLs(cfg.Port)
	go func() {
		log.Printf("ID Card Studio listening on %s", srv.Addr)
		for _, u := range urls {
			log.Printf("  %s", u)
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	if cfg.OpenBrowser {
		openURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
		if err := openBrowser(openURL); err != nil {
			log.Printf("could not open a browser automatically: %v", err)
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

// openBrowser opens url in the host's default browser.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
