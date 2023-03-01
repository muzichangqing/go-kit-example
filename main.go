package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	SERVICE_NAME = "StringService"
	SERVICE_HOST = "localhost"
	SERVICE_PORT = 8080
	CONSUL_HOST  = "127.0.0.1"
	CONSUL_PORT  = 8500
)

func main() {
	logger := log.NewLogfmtLogger(os.Stderr)

	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "my_group",
		Subsystem: "string_service",
		Name:      "request_count",
		Help:      "Number of request received",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "my_group",
		Subsystem: "string_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds",
	}, fieldKeys)
	countResult := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "my_group",
		Subsystem: "string_service",
		Name:      "count_result",
		Help:      "The result of each count method.",
	}, []string{})

	var svc StringService
	svc = stringService{}
	svc = loggingMiddleware{logger, svc}
	svc = instrumentingMiddleware{requestCount, requestLatency, countResult, svc}

	r := mux.NewRouter()
	r.Use(commonMiddleware)

	r.Methods("POST").Path("/uppercase").Handler(httptransport.NewServer(
		makeUppercaseEndpoint(svc),
		decodeUppercaseRequest,
		encodeResponse,
	))
	r.Methods("POST").Path("/count").Handler(httptransport.NewServer(
		makeCountEndpoint(svc),
		decodeCountRequest,
		encodeResponse,
	))
	r.Methods("GET").Path("/metrics").Handler(promhttp.Handler())
	r.Methods("GET").Path("/health").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("pong"))
	}))
	csr, err := NewConsulServiceRegistry(CONSUL_HOST, CONSUL_PORT, "")
	if err != nil {
		logger.Log("NewConsulServiceRegistryError", err.Error())
		return
	}
	serviceID := SERVICE_NAME + "-" + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.Itoa(rand.Intn(9000)+1000)
	if err = csr.Register(serviceID, SERVICE_NAME, SERVICE_HOST, SERVICE_PORT, false, nil); err != nil {
		logger.Log("ServiceRegisterError", err.Error())
		return
	}
	defer csr.Deregister()

	http.ListenAndServe(fmt.Sprintf("%s:%d", SERVICE_HOST, SERVICE_PORT), r)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
