
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var ctx = context.Background()

var redisSocket string
var redisPassword string
var redisHost string
var redisPort int

func init() {
	const (
		defaultPassword = ""
		usagePassword   = "Redis-Server Auth"
		defaultHost     = "localhost"
		usageHost       = "Redis-server listening IP"
		defaultPort     = 6379
		usagePort       = "Redis-server listening port"
		defaultSocket   = ""
		usageSocket     = "Redis-server listening unix-socket"
	)
	flag.StringVar(&redisPassword, "password", defaultPassword, usagePassword)
	flag.StringVar(&redisPassword, "P", defaultPassword, usagePassword+" (shorthand)")
	flag.StringVar(&redisHost, "host", defaultHost, usageHost)
	flag.StringVar(&redisHost, "h", defaultHost, usageHost+" (shorthand)")
	flag.IntVar(&redisPort, "port", defaultPort, usagePort)
	flag.IntVar(&redisPort, "p", defaultPort, usagePort+" (shorthand)")
	flag.StringVar(&redisSocket, "socket", defaultSocket, usageSocket)
	flag.StringVar(&redisSocket, "s", defaultSocket, usageSocket+" (shorthand)")
}

// var password = flag.String("password", "", "redis-server password")

func rClient() *redis.Client {
	if redisSocket != "" {
		return redis.NewClient(&redis.Options{
			Addr:     redisSocket,
			Network: "unix",
			Password: "",
			DB: 1,
		})
	} else {
		return redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
			Password: redisPassword,
		})
	}
}

func role(client *redis.Client) (interface{}, error) {
	role, err := client.Do(ctx, "role").Result()
	return role, err
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// creates a client
	client := rClient()
	// getting redis-server role status
	rawstatus, err := role(client)
	if err != nil {
		// Handle unavailable redis-server
		w.WriteHeader(http.StatusServiceUnavailable)
		currentStatus := "Unavailable"
		w.Write([]byte(fmt.Sprintf("Redis Server %s - Cannot connect\n", currentStatus)))
	} else {
		// Parse Redis-server status response
		// Response samples:
		// [master 0 []]   Master without Slave
		// [slave 127.0.0.1 6379 connected 0]  Slave of a Master Connected and synced
		// [slave 127.0.0.1 6379 connect 0]  Slave of a Master trying to connect
		// [slave 127.0.0.1 6379 sync 0]  Slave of a Master syncing
		status := rawstatus.([]interface{})
		currentRole := status[0]
		currentStatus := "Unknown"
		if currentRole == "master" {
			w.WriteHeader(http.StatusOK)
			currentStatus = "Healthy"
		} else if currentRole == "slave" {
			if status[3] == "connected" {
				w.WriteHeader(http.StatusOK)
				currentStatus = "connected"
			} else {
				currentStatus = "Unhealthy state: " + status[3].(string)
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		}
		w.Write([]byte(fmt.Sprintf("Redis %s %s\n", currentRole, currentStatus)))
	}
}

func lbStatusHandler(w http.ResponseWriter, r *http.Request) {
	// creates a client
	client := rClient()
	// getting redis-server role status
	rawstatus, err := role(client)
	if err != nil {
		// Handle unavailable redis-server
		w.WriteHeader(http.StatusServiceUnavailable)
		currentStatus := "Unavailable"
		w.Write([]byte(fmt.Sprintf("Redis Server %s - Cannot connect\n", currentStatus)))
	} else {
		// Parse Redis-server status response
		// Response samples:
		// [master 0 []]   Master without Slave
		// [slave 127.0.0.1 6379 connected 0]  Slave of a Master Connected and synced
		// [slave 127.0.0.1 6379 connect 0]  Slave of a Master trying to connect
		// [slave 127.0.0.1 6379 sync 0]  Slave of a Master syncing
		status := rawstatus.([]interface{})
		currentRole := status[0]
		currentStatus := "Not serving traffic"
		if currentRole == "master" {
			w.WriteHeader(http.StatusOK)
			currentStatus = "Healthy"
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		w.Write([]byte(fmt.Sprintf("Redis %s %s\n", currentRole, currentStatus)))
	}
}

func main() {
	flag.Parse()
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/status", statusHandler)
	r.HandleFunc("/lb_status", lbStatusHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
