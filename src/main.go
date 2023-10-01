package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	address string
	proxy   *httputil.ReverseProxy
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func NewLoadBalancer(port string, servers ...Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func HandleError(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func newSimpleServer(address string) *simpleServer {
	serverURL, err := url.Parse(address)
	HandleError(err)
	return &simpleServer{
		address: address,
		proxy:   httputil.NewSingleHostReverseProxy(serverURL),
	}
}

func (s *simpleServer) Address() string {
	return s.address
}

func (s *simpleServer) IsAlive() bool {
	return RandBool()
}

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, req)
}

// create random bool to using in isAvailable(), for acting not available
func RandBool() bool {
	return rand.Intn(2) == 1
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.twitter.com"),
		newSimpleServer("https://www.github.com"),
		newSimpleServer("https://www.google.com"),
	}

	lb := NewLoadBalancer("8000", servers...)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}

	http.HandleFunc("/", handleRedirect)
	fmt.Printf("Serving requests at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
