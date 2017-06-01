package main

import (
	"github.com/gorilla/websocket"
	"gopkg.in/mgo.v2"
	"labix.org/v2/mgo/bson"
	"net/http"
	"fmt"
	"time"
	"encoding/json"
	// "strconv"
	// "os"
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Device struct {
	Imei       string
	Timestamp int
}
type GeoJson struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// TODO
type GpsSensor struct {
	SensorId string
}

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
}
type MapMarker struct {
	Imei string `json:"imei"`
	GpsTime int `json:"gpstime"`
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
	Speed int `json:"speed"`
}
func main() {
	http.HandleFunc("/location", locationHandler)
	http.Handle("/", http.FileServer(http.Dir(".")))
	err := http.ListenAndServe(":1337", nil)
	if err != nil {
		panic("Error: " + err.Error())
	}
}

func locationHandler(w http.ResponseWriter, r *http.Request) {

	devices := make(map[string]Device)

	// TODO
	devices["356307043490167"] = Device{"356307043490167", 0}
	devices["012896004329949"] = Device{"012896004329949", 0}
	devices["013227002640237"] = Device{"013227002640237", 0}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		//log.Println(err)
		return
	}

	// TODO read from config
	mongoSession, _ := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs:    []string{"localhost"},
		Username: "username",
		Password: "password",
		Database: "gpsdb",
	})

	defer mongoSession.Close()

	c := mongoSession.DB("gpsdb").C("records")

	var record GpsRecord

	for {
		for i, device := range devices {

			// fmt.Println(i, device)

			err := c.Find(bson.M{"imei": device.Imei}).Sort("-gpstime").One(&record)

			if err != nil {
				fmt.Println("error", err)
				continue
			}

			if (record.GpsTime > device.Timestamp) {

				// gpstime := time.Unix(int64(record.GpsTime), 0)
				coor := record.Location.Coordinates

				// lon := fmt.Sprint(coor[0])
				// lat := fmt.Sprint(coor[1])
				// lat := strconv.FormatFloat(coor[1], 64)

				marker := MapMarker{record.Imei, record.GpsTime, coor[0], coor[1], record.Speed}

				// str := string({"message:[" + fmt.Sprint(gpstime) + "] " + lat+ "," + lon)

				out, err := json.Marshal(marker)
				if err != nil {
					fmt.Println("ERROR", err)
					return
				}

				err1 := conn.WriteMessage(websocket.TextMessage, []byte(out));
				if  err1 != nil {
					return
				}

				devices[i] = Device{i, record.GpsTime}

				fmt.Println(record)

			}

			time.Sleep(5000 * time.Millisecond)
		}
	}

}

