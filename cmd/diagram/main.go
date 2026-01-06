package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

//go:embed architecture.html
var content embed.FS

func main() {
	port := "9090"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := content.ReadFile("architecture.html")
		if err != nil {
			http.Error(w, "Failed to load diagram", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	url := fmt.Sprintf("http://localhost:%s", port)
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                              ║")
	fmt.Println("║  Conductor Architecture Diagram                              ║")
	fmt.Println("║                                                              ║")
	fmt.Printf("║  Server running at: %-40s ║\n", url)
	fmt.Println("║                                                              ║")
	fmt.Println("║  Press Ctrl+C to stop                                        ║")
	fmt.Println("║                                                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	go openBrowser(url)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}

