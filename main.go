package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/koding/multiconfig"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/sirupsen/logrus"
)

type config struct {
	BaseURL           string
	NClients          int
	NRequests         int
	MaxFailsPerClient int
}

func main() {
	config := &config{}
	loader := multiconfig.New()
	err := loader.Load(config)
	if err != nil {
		logrus.Fatalln("could not load config: %v", err)
	}

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		urlRequests(config)
		wg.Done()
	}()

	go func() {
		imgRequests(config)
		wg.Done()
	}()
	wg.Wait()

}

func urlRequests(config *config) {
	fmt.Println("URL/Parallel clients; total duration; avg duration per call; failed calls")
	// Init the clients
	clients := make(chan *http.Client, config.NClients)
	for i := 0; i < config.NClients; i++ {
		client := &http.Client{}
		client.Timeout = time.Second * 5
		clients <- client
	}

	totalRequests := config.NClients * config.NRequests
	maxFails := config.NClients * config.MaxFailsPerClient

	wg := sync.WaitGroup{}
	wg.Add(totalRequests)

	start := time.Now()
	nFails := 0

	// Execute
	for i := 0; i < totalRequests; i++ {
		spot := newRandomSpot()
		url := fmt.Sprintf("%s/#%v/%v/%v", config.BaseURL, spot.zoom, spot.lat, spot.lng)
		req, _ := http.NewRequest(http.MethodGet, url, nil)

		if nFails > maxFails {
			logrus.Fatalf("Failed over %v requests, aborting", maxFails)
		}

		client := <-clients
		go func(c *http.Client, req *http.Request) {
			// Return the client
			defer func() {
				time.Sleep(time.Millisecond * 100)
				clients <- client
				wg.Done()
			}()

			// execute request to render tiles
			// Executing a request can fail if the server became unreachable
			r := retrier.New(retrier.ConstantBackoff(5, 50*time.Millisecond), nil)
			var resp *http.Response
			err := r.Run(func() error {
				var err error
				resp, err = client.Do(req)
				return err
			})

			if err != nil {
				nFails++
				return
			}

			// The benchmarked request should not fail
			if resp.StatusCode != 200 {
				body, _ := ioutil.ReadAll(resp.Body)
				logrus.Fatal("Fatal code:", resp.StatusCode, ". Body:", string(body))
			}
			resp.Body.Close()

		}(client, req)
	}
	// Wait until all requests are done
	wg.Wait()

	end := time.Now()
	fmt.Printf("%v;%v;%v;%v\n", config.NClients, end.Sub(start), end.Sub(start)/time.Duration(totalRequests), nFails)
}

func imgRequests(config *config) {
	fmt.Println("IMG/Parallel clients; total duration; avg duration per call; failed calls")
	// Init the clients
	clients := make(chan *http.Client, config.NClients)
	for i := 0; i < config.NClients; i++ {
		client := &http.Client{}
		client.Timeout = time.Second * 5
		clients <- client
	}

	totalRequests := config.NClients * config.NRequests * 32
	maxFails := config.NClients * config.MaxFailsPerClient

	wg := sync.WaitGroup{}
	wg.Add(totalRequests)

	start := time.Now()
	nFails := 0

	// Execute
	for i := 0; i < totalRequests; i++ {
		url := fmt.Sprintf("%s/%s", config.BaseURL, "14/8471/5564.png")
		req, _ := http.NewRequest(http.MethodGet, url, nil)

		if nFails > maxFails {
			logrus.Fatalf("Failed over %v requests, aborting", maxFails)
		}

		client := <-clients
		go func(c *http.Client, req *http.Request) {
			// Return the client
			defer func() {
				clients <- client
				wg.Done()
			}()

			// execute request to render tiles
			// Executing a request can fail if the server became unreachable
			r := retrier.New(retrier.ConstantBackoff(5, 50*time.Millisecond), nil)
			var resp *http.Response
			err := r.Run(func() error {
				var err error
				resp, err = client.Do(req)
				return err
			})

			if err != nil {
				nFails++
				return
			}

			// The benchmarked request should not fail
			if resp.StatusCode != 200 {
				body, _ := ioutil.ReadAll(resp.Body)
				logrus.Fatal("Fatal code:", resp.StatusCode, ". Body:", string(body))
			}

			resp.Body.Close()

		}(client, req)
	}
	// Wait until all requests are done
	wg.Wait()

	end := time.Now()
	fmt.Printf("%v;%v;%v;%v\n", config.NClients, end.Sub(start), end.Sub(start)/time.Duration(totalRequests), nFails)
}

type params struct {
	zoom int
	lat  float64
	lng  float64
}

func newRandomSpot() *params {
	// zoom, random between 14, 18
	zoom := randInt(14, 22)
	// lux bbox 5.67405195478, 49.4426671413, 6.24275109216, )
	lat := randFloat(49.4426671413, 50.1280516628)
	lng := randFloat(5.67405195478, 6.24275109216)

	return &params{
		zoom: zoom,
		lat:  lat,
		lng:  lng,
	}
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
