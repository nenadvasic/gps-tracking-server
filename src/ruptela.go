/**
 * Ruptela Protocol
 */
package main

import (
	// "fmt"
	"net"
	"time"
	"log"
	"bytes"
	"encoding/binary"
	"strconv"
	"errors"
)

const (
	RUPTELA_COMMAND_RECORDS = 0x01
	RUPTELA_PROTOCOL        = "ruptela"
)

type RuptelaProtocol struct {

}

func (p *RuptelaProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string) HandlerResponse {

	res := HandlerResponse{};

	buff := bytes.NewBuffer(readbuff)

	records, err1 := p.getRecords(buff)
	if err1 != nil {
		res.error = err1
	}
	res.records = records

	// Šaljemo ACK
	_, err2 := conn.Write([]byte{0x00, 0x02, 0x64, 0x01, 0x13, 0xbc})
	if err2 != nil {
		res.error = err2
	}

	return res
}

func (p *RuptelaProtocol) getRecords(buff *bytes.Buffer) ([]GpsRecord, error) {

	var records []GpsRecord;

	var imei          uint64
	var tip           byte   // tip zahteva
	var records_left  byte   // broj preostalih recorda na uređaju (ne koristimo za sada)
	var records_count byte   // broj recorda u tekućem zahtevu
	var gpstime       uint32
	var lon           int32
	var lat           int32
	var alt           uint16
	var course        uint16
	var sat           byte
	var speed         uint16

	// buff := bytes.NewBuffer(readbuff)

	// log.Printf("%x", buff)

	buff.Next(2)

	binary.Read(buff, binary.BigEndian, &imei)
	binary.Read(buff, binary.BigEndian, &tip)

	imeiString := padLeft(strconv.FormatUint(imei, 10), "0", 15)

	// log.Println("INFO", "Device IMEI:", imeiString)

	if tip != RUPTELA_COMMAND_RECORDS {
		log.Println("ERROR", "Nepoznat tip zahteva:", tip)
		return nil, errors.New("Nepoznat tip zahteva")
	}

	binary.Read(buff, binary.BigEndian, &records_left)
	binary.Read(buff, binary.BigEndian, &records_count)

	log.Println("INFO", "Broj recorda u zahtevu:", records_count)

	for i := 0; i < int(records_count); i++ {

		binary.Read(buff, binary.BigEndian, &gpstime)

		buff.Next(2)

		binary.Read(buff, binary.BigEndian, &lon)
		binary.Read(buff, binary.BigEndian, &lat)
		binary.Read(buff, binary.BigEndian, &alt)
		binary.Read(buff, binary.BigEndian, &course)
		binary.Read(buff, binary.BigEndian, &sat)
		binary.Read(buff, binary.BigEndian, &speed)

		lon_float := float64(lon)/10000000
		lat_float := float64(lat)/10000000


		if ! isValidCoordinates(lat_float, lon_float) {
			log.Println("ERROR", "Nepravilne vrednosti koordinata! IMEI:", imeiString, "Lon:", lon_float, "Lat:", lat_float)
			continue
		}

		location := GeoJson{"Point", []float64{lon_float, lat_float}}
		sensors  := make([]GpsSensor, 0) // TODO

		buff.Next(2)

		// Senzori mogu da šalju podatke u setovima veličine 1/2/4/8 bajtova
		// Podaci su naslagani redom sa prefix bajtom koji predstavlja broj bajtova u setu (bytes_count)
		var bytes_count byte
		var sensor_id   byte
		var data1       byte
		var data2       uint16
		var data4       uint32
		var data8       uint64

		// Read 1 byte data
		binary.Read(buff, binary.BigEndian, &bytes_count)
		// fmt.Println(bytes_count)
		for j := 0; j < int(bytes_count); j++ {
			binary.Read(buff, binary.BigEndian, &sensor_id)
			binary.Read(buff, binary.BigEndian, &data1)
			// TODO: Dodavanje u slice sensors
		}

		// Read 2 byte data
		binary.Read(buff, binary.BigEndian, &bytes_count)
		// fmt.Println(bytes_count)
		for j := 0; j < int(bytes_count); j++ {
			binary.Read(buff, binary.BigEndian, &sensor_id)
			binary.Read(buff, binary.BigEndian, &data2)
			// TODO: Dodavanje u slice sensors
		}

		// Read 4 byte data
		binary.Read(buff, binary.BigEndian, &bytes_count)
		// fmt.Println(bytes_count)
		for j := 0; j < int(bytes_count); j++ {
			binary.Read(buff, binary.BigEndian, &sensor_id)
			binary.Read(buff, binary.BigEndian, &data4)
			// TODO: Dodavanje u slice sensors
		}

		// Read 8 byte data
		binary.Read(buff, binary.BigEndian, &bytes_count)
		// fmt.Println(bytes_count)
		for j := 0; j < int(bytes_count); j++ {
			binary.Read(buff, binary.BigEndian, &sensor_id)
			binary.Read(buff, binary.BigEndian, &data8)
			// TODO: Dodavanje u slice sensors
		}

		is_valid := isValidRecord(sat)

		record := GpsRecord{imeiString, location, float32(alt)/10, float32(course)/100, int(speed), int(sat), sensors, int(gpstime), int(time.Now().Unix()), RUPTELA_PROTOCOL, is_valid}

		records = append(records, record)
	}

	return records, nil
}
