package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	client := NewClient(cfg.BufferKey, cfg.BufferOrgID, cfg.MaxQueueSize)

	channels, err := client.getChannels(ctx, cfg.BufferOrgID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "channels:", err)
		os.Exit(1)
	}

	if len(channels) == 0 {
		fmt.Fprintln(os.Stderr, "No channels found. Is the organization ID correct?")
		os.Exit(1)
	}

	for _, ch := range channels {
		queue, err := client.getQueuePosts(ctx, cfg.BufferOrgID, ch.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "channel %s queue error: %v\n", ch.Name, err)
			continue
		}

		fmt.Printf("Channel %s: queue=%d\n", ch.Name, len(queue))

		if len(queue) >= cfg.MaxQueueSize {
			continue
		}

		drafts, err := client.getDraftPosts(ctx, cfg.BufferOrgID, ch.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "channel %s drafts error: %v\n", ch.Name, err)
			continue
		}

		if cfg.ShuffleDrafts {
			shufflePosts(drafts)
		}

		needed := cfg.MaxQueueSize - len(queue)
		for i := 0; i < needed && i < len(drafts); i++ {
			if err := client.pushDraftToQueue(ctx, drafts[i]); err != nil {
				fmt.Fprintf(os.Stderr, "channel %s push draft error: %v\n", ch.Name, err)
				continue
			}
			fmt.Printf("Moved draft to queue for %s: %s\n", ch.Name, drafts[i].ID)
		}
	}
}
