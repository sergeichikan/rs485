package main

import (
	"flag"
	"io"
	"log"
	"rs485/serial"
	"time"
)

const (
	rtuMinSize = 4
	rtuMaxSize = 256
	rtuExceptionSize = 5
)

var (
	address  string
	baudrate int
	databits int
	stopbits int
	enabled bool
	rtsHighDuringSend bool
	rtsHighAfterSend bool
	rxDuringTx bool
	parity   string
)

func MustClose(p io.Closer) {
	err := p.Close()
	if err != nil {
		panic(err)
	}
	log.Println("closed")
}

func ParseFlags() {
	flag.StringVar(&address, "a", "/dev/ttyUSB0", "address")
	flag.IntVar(&baudrate, "b", 9600, "baud rate")
	flag.IntVar(&databits, "d", 8, "data bits")
	flag.IntVar(&stopbits, "s", 1, "stop bits")
	flag.StringVar(&parity, "p", "N", "parity (N/E/O)")
	flag.BoolVar(&enabled, "e", false, "")
	flag.BoolVar(&rtsHighDuringSend, "r1", false, "")
	flag.BoolVar(&rtsHighAfterSend, "r2", false, "")
	flag.BoolVar(&rxDuringTx, "r3", false, "")
	flag.Parse()
}

func main() {
	ParseFlags()

	config := serial.Config{
		Address:  address,
		BaudRate: baudrate,
		DataBits: databits,
		StopBits: stopbits,
		Parity:   parity,
		Timeout:  5 * time.Second,
		RS485: serial.RS485Config{
			Enabled:            enabled,
			DelayRtsBeforeSend: 0, //time.Millisecond * 10,
			DelayRtsAfterSend:  0,
			RtsHighDuringSend:  rtsHighDuringSend,
			RtsHighAfterSend:   rtsHighAfterSend,
			RxDuringTx:         rxDuringTx,
		},
	}
	log.Println(config)
	port, err := serial.Open(&config)
	if err != nil {
		panic(err)
	}
	log.Println("connected")
	defer MustClose(port)

	var data []byte
	data = []byte{16, 4, 0, 64, 0, 1, 51, 95}
	_, err = port.Write(data)
	if err != nil {
		panic(err)
	}
	function := data[1]
	log.Println("write", data)

	data = make([]byte, 1)
	log.Println(data)

	//_, err = io.ReadFull(port, data)
	//_, err = io.ReadAtLeast(port, data, 7)
	//_, err = io.Copy(os.Stdout, port)
	//if err != nil {
	//	panic(err)
	//}

	_, err = port.Read(data)
	if err != nil {
		panic(err)
	}
	log.Println("read", data)
	log.Println(data[1] == function)
}
