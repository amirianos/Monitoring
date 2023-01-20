package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	//"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strings"
	"time"
)

var runningcontainers = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "runningcontainers",
	},
	[]string{
		"name",
		"image",
		"state",
		"status",
	},
)
var exitedcontainers = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "stopedcontainers",
	},
	[]string{
		"name",
		"image",
		"state",
		"status",
	},
)
var runningcontainersnumber = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "runningcontainersnumber",
	},
)

func update() {
	number := 0
	runningcontainers.Reset()
	exitedcontainers.Reset()
	//runningcontainersnumber.Reset()
	cli, _ := client.NewEnvClient()
	containers, _ := cli.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})
	for _, container := range containers {
		container_status := string(container.Status[:])
		container_name := strings.Join(container.Names, "")
		container_image := string(container.Image[:])
		container_state := string(container.State[:])
		container_created := container.Created
		//for running containers
		if container_state == "running" {
			runningcontainers.WithLabelValues(container_name, container_image, container_state, container_status).Set(float64(container_created))
			number += 1
		}
		//for exited containers
		if container_state == "exited" {
			exitedcontainers.WithLabelValues(container_name, container_image, container_state, container_status).Set(float64(container_created))
		}
		runningcontainersnumber.Set(float64(number))
	}
}
func main() {
	reg := prometheus.NewRegistry()
	reg.Register(exitedcontainers)
	reg.Register(runningcontainers)
	reg.Register(runningcontainersnumber)

	router := mux.NewRouter()
	router.Path("/prometheus").Handler(promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				update()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	fmt.Println("Serving requests on port 5000")
	err := http.ListenAndServe(":5000", router)
	log.Fatal(err)
}
