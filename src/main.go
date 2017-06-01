package main

import (
	// "fmt"
	// "net"
	"log"
	"os"
	"os/signal"
	"syscall"
	"encoding/json"
)

type DbConfig struct {
	Host string `json:"host"`
	User string `json:"user"`
	Pass string `json:"pass"`
	Name string `json:"name"`
	Col  string `json:"col"`
}

type Config struct {
	Host         string        `json:"host"`
	Db           *DbConfig     `json:"db"`
	GpsProtocols []GpsProtocol `json:"protocols"`
}

func main() {

	log.Println("INFO", "Program pokrenut")

	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalln("ERROR", err)
	}

	config := Config{}

	err1 := json.NewDecoder(file).Decode(&config)
	if err1 != nil {
		log.Fatalln("ERROR", err1)
	}

	count_protocols := len(config.GpsProtocols)

	log.Println("INFO", "Broj protokola u konfiguraciji:", count_protocols)

	var servers []*GpsServer;

	host := config.Host;

	for i := 0; i < count_protocols; i++ {

		protocol_name := config.GpsProtocols[i].Name
		protocol_port := config.GpsProtocols[i].Port;

		if config.GpsProtocols[i].Enabled {

			var protocol_handler GpsProtocolHandler

			if protocol_name == "ruptela" {
				protocol_handler = GpsProtocolHandler(&RuptelaProtocol{})
			} else if protocol_name == "teltonika" {
				protocol_handler = GpsProtocolHandler(&TeltonikaProtocol{})
			} else {
				log.Fatalln("ERROR", "Protocol handler nije definisan:", protocol_name)
			}

			s := NewGpsServer(protocol_name, config.Db, protocol_handler)

			s.Start(host, protocol_port)

			// log.Println("INFO", "Server pokrenut za protokol " + protocol_name + " na portu " + protocol_port)

			servers = append(servers, s);
		}
	}

	log.Println("INFO", "Svi serveri su pokrenuti")

	// os.Exit(0);

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	// <-ch
	log.Println("INFO", "Dobijen signal", <-ch)

	stopServers(servers)

	// time.Sleep(10000 * time.Millisecond)
	log.Println("INFO", "Program zaustavljen")
}

func stopServers(servers []*GpsServer) {

	for _, server := range servers {
		server.Stop()
	}
}

// HELPERS

func padLeft(str, pad string, lenght int) string {

	if len(str) >= lenght {
		return str
	}

	for {
		str = pad + str
		if len(str) >= lenght {
			return str
		}
	}
}





