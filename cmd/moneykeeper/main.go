package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"moneykeeper/internal/bot"
	"moneykeeper/internal/chain"
	"moneykeeper/internal/fx"
	"moneykeeper/internal/notify"
	"moneykeeper/internal/pgclient"
	"moneykeeper/internal/repo"
	"moneykeeper/internal/service"
	"moneykeeper/internal/telegram"
	"moneykeeper/internal/web"
	"moneykeeper/internal/web/appleauth"
)

const (
	httpAddr        = ":8090"
	shutdownTimeout = 10 * time.Second
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("exit", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return errors.New("TELEGRAM_BOT_TOKEN is not set")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is not set")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgclient.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer db.Close()

	svc := service.New(db, repo.NewUsers(db), repo.NewBudgets(db), repo.NewAccounts(db),
		repo.NewCategories(db), repo.NewTransactions(db), repo.NewPlan(db), repo.NewSchedules(db), repo.NewRates(db), repo.NewAssets(db), fx.NewProvider(), log)
	svc.WithImportRules(repo.NewImportRules(db))
	svc.WithSessions(repo.NewSessions(db))
	svc.WithWallets(repo.NewWallets(db), chain.NewClient())
	if err := svc.LoadAssets(ctx); err != nil {
		return fmt.Errorf("load assets: %w", err)
	}
	// Напоминания о платежах уходят из этого же процесса.
	tg := telegram.New(token)
	go notify.New(svc, tg, log).Run(ctx)
	// Бот принимает сообщения в личном чате и записывает из них траты.
	go bot.New(svc, tg, log).Run(ctx)

	// Вход через Apple для нативного приложения: без APPLE_CLIENT_ID (bundle id) он отвечает «недоступен».
	var apple web.AppleVerifier
	if id := os.Getenv("APPLE_CLIENT_ID"); id != "" {
		apple = appleauth.New(id)
	}

	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           web.NewHandler(svc, token, log, apple),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Info("listening", "addr", httpAddr)

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
