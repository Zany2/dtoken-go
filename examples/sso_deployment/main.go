package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/sso"
	redisstore "github.com/Zany2/dtoken-go/storage/redis"
)

func main() {
	role := flag.String("role", "", "center or client")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *role); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

// run owns shared storage until HTTP requests drain and component cleanup completes. run 持有共享存储，直到 HTTP 请求结束且组件清理完成。
func run(ctx context.Context, role string) error {
	cfg, err := loadConfig(role, os.Getenv)
	if err != nil {
		return err
	}
	var storage contract.Storage = sso.NewMemoryStorage()
	if cfg.redisURL != "" {
		adapter, err := redisstore.NewStorage(cfg.redisURL)
		if err != nil {
			return errors.New("SSO Redis initialization failed; check address, credentials and connectivity")
		}
		defer adapter.Close()
		storage = adapter
	}
	app, err := newApplication(cfg, storage, nil)
	if err != nil {
		return err
	}
	defer app.close()
	server := &http.Server{Addr: cfg.listen, Handler: app.handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	log.Printf("SSO %s listening on %s; public origin %s", cfg.role, cfg.listen, cfg.publicURL)
	return serve(ctx, server)
}

// serve stops admission and waits for handlers after connection closure so storage outlives in-flight operations. serve 停止接收请求并在连接关闭后等待处理器返回，确保存储生命周期覆盖在途操作。
func serve(ctx context.Context, server *http.Server) error {
	var active sync.WaitGroup
	var mu sync.Mutex
	accepting := true
	handler := server.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if !accepting {
			mu.Unlock()
			http.Error(w, "server shutting down", http.StatusServiceUnavailable)
			return
		}
		active.Add(1)
		mu.Unlock()
		defer active.Done()
		handler.ServeHTTP(w, r)
	})
	stopAdmission := func() { mu.Lock(); accepting = false; mu.Unlock() }
	defer func() {
		stopAdmission()
		_ = server.Close()
		active.Wait()
	}()

	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		stopAdmission()
		deadline, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := server.Shutdown(deadline); err != nil {
			_ = server.Close()
			<-result
			return errors.New("HTTP graceful shutdown timed out")
		}
		err := <-result
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
