package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/gosuri/uilive"
	"github.com/rs/zerolog"
	"github.com/ssgreg/repeat"
	"github.com/urfave/cli/v2"
)

type Handler struct {
	logger zerolog.Logger
	writer *uilive.Writer
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	hdl := NewHandler(logger)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:     "conn",
			Required: true,
			Usage:    "Connection name",
		},
		&cli.StringFlag{
			Name:     "ping",
			Required: true,
			Usage:    "IP to ping",
		},
	}

	app := &cli.App{
		Flags: flags,
		Action: func(cCtx *cli.Context) error {
			return hdl.Start(cCtx.Context, cCtx.String("conn"), cCtx.String("ping"))
		},
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		logger.Error().Err(err).Msg("run fail")
	}
}

func NewHandler(logger zerolog.Logger) *Handler {
	writer := uilive.New()

	return &Handler{
		logger: logger,
		writer: writer,
	}
}

func (hdl *Handler) Start(ctx context.Context, conn, ip string) error {
	hdl.writer.Start()

	hdl.connect(ctx, conn)

	ticker := time.NewTicker(time.Second * 10)

	fails := 0

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			_ = hdl.disconnect(ctx)

			hdl.writer.Stop()

			return nil
		case <-ticker.C:
			if err := hdl.ping(ctx, ip); err != nil {
				fails++

				fmt.Fprintf(hdl.writer, "Status: fail, pings %d\n", fails)
			} else {
				fails = 0

				fmt.Fprintln(hdl.writer, "Status: ok")
			}

			if fails == 10 {
				_ = hdl.disconnect(ctx)

				hdl.connect(ctx, conn)
			}
		}
	}
}

func (hdl *Handler) connect(ctx context.Context, conn string) {
	fmt.Fprintln(hdl.writer, "Connecting...")

	delay := &repeat.ExponentialBackoffBuilder{}

	_ = repeat.Repeat(
		repeat.Fn(func() error {
			cmd := exec.CommandContext(ctx, "nmcli", "connection", "up", conn)

			if err := cmd.Run(); err != nil {
				return repeat.HintTemporary(err)
			}

			return nil
		}),
		repeat.StopOnSuccess(),
		repeat.WithDelay(
			repeat.SetContext(ctx),
			delay.WithInitialDelay(time.Second*10).WithMaxDelay(time.Minute).
				WithMultiplier(2).WithJitter(0.5).Set(),
		),
	)

	fmt.Fprintln(hdl.writer, "Connected")
}

func (hdl *Handler) disconnect(ctx context.Context) error {
	fmt.Fprintln(hdl.writer, "Disconnecting...")

	cmd := exec.CommandContext(ctx, "nmcli", "connection", "down", "Office")

	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Fprintln(hdl.writer, "Disconnected")

	return nil
}

func (hdl *Handler) ping(ctx context.Context, ip string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", ip)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
