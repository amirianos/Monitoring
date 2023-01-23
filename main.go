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
	"regexp"
	//"reflect"
	"net/http"
	"os/exec"
	"strconv"
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
var ramusage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "ramusage",
	},
	[]string{
		"containername",
	},
)
var cpuusage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "cpuusage",
	},
	[]string{
		"containername",
	},
)

var command1 string = "docker"
var command2 string = "stats"
var command3 string = "--no-stream"
var command4 string = "--all"

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
		container_names := strings.Split(container_name, "/")

		if container_state == "running" {
			runningcontainers.WithLabelValues(container_names[1], container_image, container_state, container_status).Set(float64(container_created))
			number += 1
		}
		//for exited containers
		if container_state == "exited" {
			exitedcontainers.WithLabelValues(container_names[1], container_image, container_state, container_status).Set(float64(container_created))
		}
		runningcontainersnumber.Set(float64(number))
	}

	output, err := exec.Command(command1, command2, command3, command4).Output()
	if err != nil {
		fmt.Println("you have some errors")
	}
	containers_status := strings.Split(string([]byte(output[:])), "\n")
	containers_status = containers_status[1 : len(containers_status)-1]
	cpuusage.Reset()
	ramusage.Reset()
	for _, cont := range containers_status {
		cont1 := strings.Split(cont, " ")
		number = 0
		name := "a"
		ram_usage := "a"
		cpu_usage := "a"
		for j := 0; j < len(cont1); j++ {
			if len(cont1[j]) == 0 {
				continue
			}
			number++
			if number == 2 {
				name = cont1[j]
			}
			if number == 3 {
				cpu_usage = cont1[j]
			}
			if number == 7 {
				ram_usage = cont1[j]
			}
		}
		re := regexp.MustCompile(`(\d{1,4}).(\d{1,2})`)
		ram_usage_float, _ := strconv.ParseFloat(re.FindString(ram_usage), 64)
		cpu_usage_float, _ := strconv.ParseFloat(re.FindString(cpu_usage), 64)
		ramusage.WithLabelValues(name).Set(ram_usage_float)
		cpuusage.WithLabelValues(name).Set(cpu_usage_float)
	}

}
func main() {
	reg := prometheus.NewRegistry()
	reg.Register(exitedcontainers)
	reg.Register(runningcontainers)
	reg.Register(runningcontainersnumber)
	reg.Register(ramusage)
	reg.Register(cpuusage)

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
