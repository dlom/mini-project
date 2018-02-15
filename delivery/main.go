package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"os"
)

func main() {
	redisHost := os.Getenv("REDIS_URL")
	if redisHost == "" {
		redisHost = "redis://localhost:6379"
	}

	fmt.Println("Attempting to connect to redis...")
	c, err := redis.DialURL(redisHost)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Failed to connect to redis!")
		fmt.Fprintln(os.Stderr, "Retrying... (probably)")
		os.Exit(1)
	}
	fmt.Println("Connected!")
	defer c.Close()

	for {
		s, err := redis.Strings(c.Do("BLPOP", "queue:postback", 0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Printf("%#v\n", s[1])
	}
}
