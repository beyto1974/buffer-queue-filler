package main

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	MaxQueueSize  int
	BufferOrgID   string
	BufferKey     string
	ShuffleDrafts bool
}

func loadConfig() Config {
	var cfg Config

	flag.IntVar(&cfg.MaxQueueSize, "maxQueueSize", 10, "maximum queue size before stopping")
	flag.StringVar(&cfg.BufferOrgID, "bufferOrgId", "", "Buffer organization ID")
	flag.StringVar(&cfg.BufferKey, "bufferKey", "", "Buffer API key")
	flag.BoolVar(&cfg.ShuffleDrafts, "shuffleDrafts", false, "shuffle drafts before queueing")
	flag.Parse()

	if cfg.MaxQueueSize < 1 {
		fmt.Fprintln(os.Stderr, "maxQueueSize must be >= 1")
		os.Exit(1)
	}
	if cfg.BufferOrgID == "" {
		fmt.Fprintln(os.Stderr, "bufferOrgId is required")
		os.Exit(1)
	}
	if cfg.BufferKey == "" {
		fmt.Fprintln(os.Stderr, "bufferKey is required")
		os.Exit(1)
	}

	return cfg
}
