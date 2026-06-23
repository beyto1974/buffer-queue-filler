package main

import (
	"math/rand"
	"time"
)

func shufflePosts(posts []Post) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(posts), func(i, j int) {
		posts[i], posts[j] = posts[j], posts[i]
	})
}
