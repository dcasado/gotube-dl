package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	listenAddressEnvVariable = "LISTEN_ADDRESS"
	listenPortEnvVariable    = "LISTEN_PORT"
)

type downloadBody struct {
	URL string `json:"url"`
}

func main() {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("POST /download", func(w http.ResponseWriter, r *http.Request) {
		var b downloadBody
		d := json.NewDecoder(r.Body)
		err := d.Decode(&b)
		if err != nil {
			http.Error(w, "Failed to parse JSON body", http.StatusBadRequest)
			return
		}
		if b.URL == "" {
			http.Error(w, "URL field cannot be empty", http.StatusBadRequest)
			return
		}

		cmd := exec.Command("yt-dlp", b.URL)
		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb

		log.Printf("Starting download of %s", b.URL)
		err = cmd.Run()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/text")
			w.Write([]byte(fmt.Sprintf("Failed to download video: %s", errb.String())))
			return
		}

		lines := strings.Split(outb.String(), "\n")
		for _, line := range lines {
			fmt.Println(line)
		}
		log.Printf("Download of %s is complete", b.URL)
		w.WriteHeader(http.StatusOK)
	})

	serveMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte("Ok"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", getListenAddress(), getListenPort()),
		Handler: serveMux,
	}

	// Handle gracefull shutdown
	errC := make(chan error, 1)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-ctx.Done()

		log.Println("Shutdown signal received")

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer func() {
			stop()
			cancel()
			close(errC)
		}()

		server.SetKeepAlivesEnabled(false)

		if err := server.Shutdown(ctxTimeout); err != nil {
			errC <- err
		}

		log.Println("Shutdown completed")
	}()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting the server: %s", err)
		}
	}()
	log.Printf("Started server listening on %s:%s", getListenAddress(), getListenPort())

	if err := <-errC; err != nil {
		log.Fatalln("error", err)
	}
	log.Print("Exited properly")
}

func getListenAddress() string {
	return optionalVariable(listenAddressEnvVariable, "127.0.0.1")
}

func getListenPort() string {
	return optionalVariable(listenPortEnvVariable, "8080")
}

func optionalVariable(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}
