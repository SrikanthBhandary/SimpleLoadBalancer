package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/gorilla/mux"
	"github.com/jasonlvhit/gocron"
)

type loadbalancerConfig struct {
	Hosts []struct {
		Host    string   `yaml:"host"`
		Servers []string `yaml:"servers"`
	} `yaml:"hosts"`
	Paths []struct {
		Path    string   `yaml:"path"`
		Servers []string `yaml:"servers"`
	} `yaml:"paths"`
}

// Server holds the type required for Server
type Server struct {
	mu       sync.Mutex
	Endpoint string
	Path     string
	Healthy  bool
	Scheme   string
}

// HealthCheck checks the health of the server
func (s *Server) HealthCheck() {
	targetURL := s.Scheme + s.Endpoint + s.Path
	response, err := http.Get(targetURL)
	if err != nil {
		s.mu.Lock()
		s.Healthy = false
		defer s.mu.Unlock()
		return
	}

	if response.StatusCode != http.StatusOK {
		s.mu.Lock()
		s.Healthy = false
		defer s.mu.Unlock()
	} else {
		s.Healthy = true
	}
}

var register = make(map[string][]*Server)
var config loadbalancerConfig

func transformBackends() {
	for _, item := range config.Paths {
		for _, server := range item.Servers {
			register[item.Path] = append(register[item.Path], &Server{Endpoint: server, Path: "/healthcheck", Healthy: true, Scheme: "http://"})
		}
	}
	for _, item := range config.Hosts {
		for _, server := range item.Servers {
			register[item.Host] = append(register[item.Host], &Server{Endpoint: server, Path: "/healthcheck", Healthy: true, Scheme: "http://"})
		}
	}
}

func healthyServers(hostOrPath string) []*Server {
	var healthyServerEndpoints []*Server
	for i := range register[hostOrPath] {
		item := register[hostOrPath][i]
		item.mu.Lock()
		if item.Healthy == true {
			healthyServerEndpoints = append(healthyServerEndpoints, item)
		}
		defer item.mu.Unlock()
	}
	return healthyServerEndpoints
}

// HomePage handler is the default place where all the requests are rerouted.
func HomePage(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	tagetURL := ""
	for _, item := range config.Hosts {
		if host == item.Host {
			// targetServers := item.Servers
			targetServers := healthyServers(item.Host)
			if len(targetServers) == 0 {
				w.Write([]byte("Internals server error"))
				return
			}
			randomIndex := rand.Intn(len(targetServers))
			pick := targetServers[randomIndex].Endpoint
			tagetURL = fmt.Sprintf("http://%s", pick)
			break
		} else {
			continue
		}
	}
	response, err := http.Get(tagetURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal sever error"))
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}
		w.Write([]byte(contents))
	}
}

func pathHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	tagetURL := ""
	for _, item := range config.Paths {
		if path == item.Path {
			// targetServers := item.Servers
			targetServers := healthyServers(item.Path)
			if len(targetServers) == 0 {
				w.Write([]byte("Internals server error"))
				return
			}
			randomIndex := rand.Intn(len(targetServers))
			pick := targetServers[randomIndex].Endpoint
			tagetURL = fmt.Sprintf("http://%s", pick)
			break
		} else {
			continue
		}
	}
	response, err := http.Get(tagetURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal sever error"))
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}
		w.Write([]byte(contents))
	}
}

func myTask() {
	for key := range register {
		for i := range register[key] {
			fmt.Printf("%p -- %s --- %t \n", register[key][i], register[key][i].Endpoint, register[key][i].Healthy)
			server := register[key][i]
			server.HealthCheck()
		}
	}
}

func periodicHealthCheck() {
	gocron.Every(10).Second().Do(myTask)
	<-gocron.Start()
}

func main() {
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("Error reading config file: %s\n", err)
		return
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Printf("Error parsing config file: %s\n", err)
		os.Exit(1)
	}
	transformBackends()
	go periodicHealthCheck()
	m := mux.NewRouter()
	m.HandleFunc("/", HomePage).Methods("GET")
	m.HandleFunc(`/{\.*}`, pathHandler)
	log.Fatal(http.ListenAndServe(":8000", m))
}
