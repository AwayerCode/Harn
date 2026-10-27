package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/AwayerCode/Harn/apps/swe-cli/internal/systemdesign"
)

func systemDesignPreview(args []string, out io.Writer) int {
	fail := func(err error) int { return encode(out, systemdesign.UnavailableReport("preview", Version, err), 2) }
	f := flag.NewFlagSet("system-design preview", flag.ContinueOnError)
	root := f.String("root", ".", "base for a relative YAML path")
	file := f.String("file", "", "system-design.yaml path")
	expected := f.String("expect-version", "", "required exact CLI version")
	help := setHelp(f, out, "swe-cli system-design preview --file <system-design.yaml> --expect-version <version>\nServes fresh CLI output on a random loopback port for up to 30 minutes. Ctrl-C stops the preview.")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return help()
		}
		return fail(err)
	}
	if f.NArg() != 0 || *file == "" {
		return fail(errors.New("preview requires --file and accepts no positional arguments"))
	}
	if *expected == "" || *expected != Version {
		return fail(fmt.Errorf("--expect-version must match this CLI build (%s)", Version))
	}
	path := *file
	if !filepath.IsAbs(path) {
		path = filepath.Join(*root, path)
	}
	preview, report, code := systemdesign.NewPreview(path, Version)
	if code != 0 {
		return encode(out, report, code)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fail(err)
	}
	defer listener.Close()
	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(stopCtx, 30*time.Minute)
	defer cancel()
	return serveDesignPreview(ctx, listener, preview, out)
}

func serveDesignPreview(ctx context.Context, listener net.Listener, preview *systemdesign.Preview, out io.Writer) int {
	host := listener.Addr().String()
	server := &http.Server{Handler: preview.Handler(host), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	if code := encode(out, map[string]any{"format": "swe-preview-v1", "url": "http://" + host + preview.Path(), "source": preview.File(), "generator_version": Version, "lifetime_seconds": 1800}, 0); code != 0 {
		_ = server.Close()
		<-result
		return 2
	}
	select {
	case err := <-result:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return encode(out, systemdesign.UnavailableReport("preview", Version, err), 2)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
		}
		<-result
	}
	return 0
}
