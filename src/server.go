/**
 * GPS Tracking Server
 */
package main

import (
	"sync"
	"log"
	"net"
	"gopkg.in/mgo.v2"
	"time"
	// "errors"
)

type GpsProtocolHandler interface {
	handle([]byte, *net.TCPConn, string) HandlerResponse
}

type HandlerResponse struct {
	error   error
	imei    string
	records []GpsRecord
}

type GpsProtocol struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Port    string `json:"port"`
	Enabled bool   `json:"enabled"`
}

type GeoJson struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// TODO
type GpsSensor struct {
	SensorId string
}

// type GpsDevice struct {
// 	Imei      string
// 	IpAddress string
// 	// Active bool
// }

type GpsRecord struct {
	Imei       string      `json:"imei"`
	Location   GeoJson     `json:"location"`
	//         Latitude    float64             `json:"lat"`
	//         Longitude   float64             `json:"lon"`
	Altitude   float32     `json:"alt"`
	Course     float32     `json:"course"`
	Speed      int         `json:"speed"`
	Satellites int         `json:"satellites"`
	Sensors    []GpsSensor `json:"sensors"`
	GpsTime    int         `json:"gpstime"`    // vreme dobijeno od uređaja
	Timestamp  int         `json:"timestamp"`
	Protocol   string      `json:"protocol"`
	Valid      bool        `json:"valid"` // Zapis smatramo validnim ako ima 3+ satelita
}

type GpsServer struct {
	name         string
	ch           chan bool
	waitGroup    *sync.WaitGroup
	dbConfig     *DbConfig
	mongoSession *mgo.Session
	listener     *net.TCPListener
	protocol     GpsProtocolHandler
}

func NewGpsServer(name string, dbConfig *DbConfig, protocol GpsProtocolHandler) *GpsServer {

	// log.Fatalln(protocol)

	log.Println("INFO", "Inicijalizacija servera:", name)

	mongoSession, err := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs:    []string{dbConfig.Host},
		Username: dbConfig.User,
		Password: dbConfig.Pass,
		Database: dbConfig.Name,
	})
	// sessionCopy.SetMode(mgo.Monotonic, true)

	if err != nil {
		log.Fatalln("ERROR", "Neuspešno konektovanje na bazu:", err)
	}

	s := &GpsServer{
		name:         name,
		ch:           make(chan bool),
		waitGroup:    &sync.WaitGroup{},
		dbConfig:     dbConfig,
		mongoSession: mongoSession,
		protocol:     protocol,
	}

	s.waitGroup.Add(1)
	return s
}

func (s *GpsServer) Start(host string, port string) {

	log.Println("INFO", "Pokretanje servera:", s.name) // + " on [" + host + ":" + port + "] ...")
	s.Listen(host, port)
	go s.Serve()
}

func (s *GpsServer) Listen(host string, port string) {

	laddr, _ := net.ResolveTCPAddr("tcp", host + ":" + port)

	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		log.Fatalln("ERROR", "Program nije u mogućnosti da otvori listening socket:", err.Error())
	}
	// defer listener.Close()

	log.Println("INFO", "Socket uspešno otvoren na", listener.Addr())

	s.listener = listener
}

func (s *GpsServer) Serve() {
	defer s.waitGroup.Done()
	for {
		select {
			case <-s.ch:
				log.Println("INFO", "Stopping listening on", s.listener.Addr())
				s.listener.Close()
				return
			default:
		}
		s.listener.SetDeadline(time.Now().Add(5e9)) // 5 secs
		conn, err := s.listener.AcceptTCP()
		if nil != err {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			log.Println(err)
		}
		log.Println("INFO", "Connected:", conn.RemoteAddr())
		// conn.SetKeepAlive(true)
		s.waitGroup.Add(1)
		go s.HandleRequest(conn)
	}
}

func (s *GpsServer) HandleRequest(conn *net.TCPConn) {

	defer log.Println("INFO", "Disconnecting:", conn.RemoteAddr())
	defer conn.Close()
	defer s.waitGroup.Done()

	var imei string

	var i int = 0

	for {
		select {
			case <-s.ch:
				return
			default:
		}

		conn.SetReadDeadline(time.Now().Add(5e9)) // 5 secs

		readbuff := make([]byte, 2048) // TODO: Šta ako je input veći od 2048 ?

		_, err := conn.Read(readbuff)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				// Posle 60 timeout-a (5 minuta) zatvaramo konekciju
				if i >= 60 {
					return
				} else {
					i++
					continue
				}
			}
			log.Println("ERROR", "Connection Read:", err)
			return
		} else {
			i = 0
		}

		res := s.protocol.handle(readbuff, conn, imei)
		if res.error != nil {
			log.Println("ERROR", res.error);
			return
		}

		imei = res.imei

		if len(res.records) > 0 {
			s.SaveGpsRecords(res.records)
		}
	}
}

func (s *GpsServer) SaveGpsRecords(records []GpsRecord) bool {

	sessionCopy := s.mongoSession.Copy()
	defer sessionCopy.Close()

	c := sessionCopy.DB(s.dbConfig.Name).C(s.dbConfig.Col)

	for _, record := range records {
		err1 := c.Insert(record)
		if err1 != nil {
			log.Println("ERROR", "Neuspešan upis recorda u bazu:", err1)
			return false
		}

		log.Println("INFO", "Record sačuvan", record.Imei, record.Location.Coordinates, record.Speed, record.Sensors, time.Unix(int64(record.GpsTime), 0), record.Protocol)
	}

	return true
}

func (s *GpsServer) Stop() {
	log.Println("INFO", "Zaustavljanje servera:", s.name)
	// s.mongoSession.Close()
	// s.listener.Close()
	close(s.ch)
	s.waitGroup.Wait()
}

func isValidCoordinates(lat_float float64, lon_float float64) bool {
	if lon_float == 0 || lon_float < -180 || lon_float > 180 || lat_float == 0 || lat_float < -90 || lat_float > 90 {
		return false
	}
	return true
}

func isValidRecord(satellites byte) bool {
	return (satellites > 3)
}


/*
func writeLog(a ...interface{}) {

	time := time.Now().Format("2006-01-02 15:04:05")

	fmt.Print("[", time, "] ")
	fmt.Println(a...)
}
*/
