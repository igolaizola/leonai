package leonai

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/igolaizola/leonai/pkg/leonardo"
)

type Config struct {
	Proxy  string
	Wait   time.Duration
	Debug  bool
	Cookie string
}

// Run runs the leonai process.
func GenerateVideo(ctx context.Context, cfg *Config, image string, motionStrength int, output string) error {
	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
	}
	if cfg.Proxy != "" {
		u, err := url.Parse(cfg.Proxy)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(u),
		}
	}
	client := leonardo.New(&leonardo.Config{
		Wait:        cfg.Wait,
		Debug:       cfg.Debug,
		Client:      httpClient,
		CookieStore: leonardo.NewCookieStore(cfg.Cookie),
	})
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("couldn't start leonardo client: %w", err)
	}
	defer func() {
		if err := client.Stop(ctx); err != nil {
			log.Printf("couldn't stop leonardo client: %v\n", err)
		}
	}()
	imageID, err := client.Upload(ctx, image)
	if err != nil {
		return fmt.Errorf("couldn't upload image: %w", err)
	}
	id, u, err := client.CreateMotion(ctx, imageID, motionStrength)
	if err != nil {
		return fmt.Errorf("couldn't create motion: %w", err)
	}
	log.Println("id:", id)
	log.Println("url:", u)
	if output != "" {
		if err := download(ctx, httpClient, u, output); err != nil {
			return fmt.Errorf("couldn't download video: %w", err)
		}
	}
	return nil
}

func download(ctx context.Context, client *http.Client, url, output string) error {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("couldn't create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't download video: %w", err)
	}
	defer resp.Body.Close()

	// Write video to output
	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("couldn't create temp file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("couldn't write to temp file: %w", err)
	}
	return nil
}
