package main

import (
	"fmt"
	"net/http"

	"idrac-exporter/chassis"
	"idrac-exporter/config"
	"idrac-exporter/system"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stmcginnis/gofish"
)

func metrichandler(w http.ResponseWriter, r *http.Request) {
	// var err error
	conf := gofish.ClientConfig{
		Endpoint: r.URL.Query().Get("idrac_host"),
		// Endpoint: r.URL.Query().Get("localhost:8080/static/test.json"),
		Username: config.Idracuser,
		Password: config.Idracpassword,
		Insecure: true,
	} // struct connection Redfish Service: url + username + password + authen

	fmt.Println(r.URL.Query().Get("idrac_host"))
	// fmt.Println(r.URL.Query().Get("localhost:8080/static/test.json"))

	var err error
	config.GOFISH, err = gofish.Connect(conf) // kết nối đến Redfish service của server

	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	defer config.GOFISH.Logout()

	fmt.Println(" Connect successful")

	mhandler := promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		})
	mhandler.ServeHTTP(w, r)
}

func main() {
	const PORT = "9099"
	fmt.Println("Server listening at ", PORT)

	// Listen all interfaces at port 9000
	const IP_ADDRESS = ":" + PORT

	system := system.SystemCollector{}
	prometheus.Register(system) // func Register

	chassis := chassis.Chassis{}
	prometheus.Register(chassis)

	// Starting server
	http.HandleFunc("/metrics", metrichandler)
	http.ListenAndServe(IP_ADDRESS, nil)
}
