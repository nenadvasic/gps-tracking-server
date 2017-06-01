/**
 * Teltonika Protocol
 */
package main

import (
	// "fmt"
	"net"
	"time"
	"log"
	"bytes"
	"encoding/binary"
	// "strconv"
	"errors"
)

const (
	TELTONIKA_PROTOCOL      = "teltonika"
	TELTONIKA_CODEC_GH3000  = 0x07
	TELTONIKA_CODEC_FM4X00  = 0x08
	TELTONIKA_CODEC_12      = 0x0C

)

type TeltonikaProtocol struct {

}

func (p *TeltonikaProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string) HandlerResponse {

	buff := bytes.NewBuffer(readbuff)

	var start_bytes uint16

	binary.Read(buff, binary.BigEndian, &start_bytes)

	res := HandlerResponse{};

	// Ako imamo nešto u prva 2 bajta onda je uređaj poslao svoj IMEI
	if start_bytes > 0 {

		res.imei = p.getIMEI(buff)

		log.Println("INFO", "Device IMEI:", res.imei)

		_, err := conn.Write([]byte{0x01}) // ACK
		if err != nil {
			res.error = err
		}
	// Ako su prva dva bajta nule onda je uređaj poslao GPS zapise
	} else {

		res.imei = imei

		records, err1 := p.getRecords(buff, imei)
		if err1 != nil {
			res.error = err1
		}
		res.records = records

		// Šaljemo ACK
		err2 := binary.Write(conn, binary.BigEndian, int32(len(records)))
		if err2 != nil {
			res.error = err2
		}
	}

	return res
}

func (p *TeltonikaProtocol) getIMEI(buff *bytes.Buffer) string {

	var imei string

	buff.Truncate(15)

	imei = buff.String()

	if imei == "" {
		// TODO ?
	}

	return padLeft(imei, "0", 15)
}

func (p *TeltonikaProtocol) getRecords(buff *bytes.Buffer, imei string) ([]GpsRecord, error) {

	var records []GpsRecord;

	var data_length   uint32
	var codec         byte
	var priority      byte   // ne koristimo za sada
	var records_count byte   // broj recorda u tekućem zahtevu
	var gpstime       uint64
	var lon           int32
	var lat           int32
	var alt           int16
	var course        uint16
	var sat           byte
	var speed         uint16

	buff.Next(2)

	binary.Read(buff, binary.BigEndian, &data_length)
	binary.Read(buff, binary.BigEndian, &codec)

	if codec == TELTONIKA_CODEC_12 {
		// TODO ?
		return nil, errors.New("CODEC 12")
	}

	binary.Read(buff, binary.BigEndian, &records_count)

	log.Println("INFO", "Broj recorda u zahtevu:", records_count)

	for i := 0; i < int(records_count); i++ {

		if (codec == TELTONIKA_CODEC_GH3000) {
			// TODO
		} else {
			binary.Read(buff, binary.BigEndian, &gpstime)
			binary.Read(buff, binary.BigEndian, &priority)

			binary.Read(buff, binary.BigEndian, &lon)
			binary.Read(buff, binary.BigEndian, &lat)
			binary.Read(buff, binary.BigEndian, &alt)
			binary.Read(buff, binary.BigEndian, &course)
			binary.Read(buff, binary.BigEndian, &sat)
			binary.Read(buff, binary.BigEndian, &speed)

			lon_float := float64(lon)/10000000
			lat_float := float64(lat)/10000000

			if ! isValidCoordinates(lat_float, lon_float) {
				log.Println("ERROR", "Nepravilne vrednosti koordinata! IMEI:", imei, "Lon:", lon_float, "Lat:", lat_float)
				continue
			}

			location := GeoJson{"Point", []float64{lon_float, lat_float}}
			sensors  := make([]GpsSensor, 0) // TODO

			is_valid := isValidRecord(sat)

			record := GpsRecord{imei, location, float32(alt)/10, float32(course)/100, int(speed), int(sat), sensors, int(gpstime/1000), int(time.Now().Unix()), TELTONIKA_PROTOCOL, is_valid}

			// log.Println(record)

			records = append(records, record)

			buff.Next(6) // TODO senzori
		}
	}

	return records, nil
}


