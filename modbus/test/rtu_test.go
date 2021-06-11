package test

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"rs485/modbus"
	"testing"
)

var address string
var baudrate int
var databits int
var stopbits int
var parity   string
var slaveId uint
var message string

func ParseFlags() {
	flag.StringVar(&address, "a", "/dev/ttyUSB0", "address")
	flag.IntVar(&baudrate, "b", 9600, "baud rate")
	flag.IntVar(&databits, "d", 8, "data bits")
	flag.IntVar(&stopbits, "s", 1, "stop bits")
	flag.StringVar(&parity, "p", "N", "parity (N/E/O)")
	flag.UintVar(&slaveId, "slave_id", 16, "slave id")
	flag.StringVar(&message, "m", "serial", "message")
	flag.Parse()
}

func MustClose(p io.Closer) {
	err := p.Close()
	if err != nil {
		panic(err)
	}
	log.Println("closed")
}

func NewRTUClientHandler(address string) *modbus.RTUClientHandler {
	handler := modbus.NewRTUClientHandler(address)
	handler.BaudRate = baudrate
	handler.DataBits = databits
	handler.Parity = parity
	handler.StopBits = stopbits
	handler.SlaveId = byte(slaveId)
	handler.Logger = log.New(os.Stdout, "rtu: ", log.LstdFlags)
	return handler
}

func dataBlock(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

func responseError(response *modbus.ProtocolDataUnit) error {
	mbError := &modbus.ModbusError{FunctionCode: response.FunctionCode}
	if response.Data != nil && len(response.Data) > 0 {
		mbError.ExceptionCode = response.Data[0]
	}
	return mbError
}

func send(mb *modbus.RTUClientHandler, request *modbus.ProtocolDataUnit) (response *modbus.ProtocolDataUnit, err error) {
	aduRequest, err := mb.Encode(request)
	if err != nil {
		return
	}
	aduResponse, err := mb.Send(aduRequest)
	if err != nil {
		return
	}
	if err = mb.Verify(aduRequest, aduResponse); err != nil {
		return
	}
	response, err = mb.Decode(aduResponse)
	if err != nil {
		return
	}
	// Check correct function code returned (exception)
	if response.FunctionCode != request.FunctionCode {
		err = responseError(response)
		return
	}
	if response.Data == nil || len(response.Data) == 0 {
		// Empty response
		err = fmt.Errorf("modbus: response data is empty")
		return
	}
	return
}

func ReadInputRegisters(mb *modbus.RTUClientHandler, address, quantity uint16) (results []byte, err error) {
	if quantity < 1 || quantity > 125 {
		err = fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v',", quantity, 1, 125)
		return
	}
	request := modbus.ProtocolDataUnit{
		FunctionCode: modbus.FuncCodeReadInputRegisters,
		Data:         dataBlock(address, quantity),
	}
	response, err := send(mb, &request)
	if err != nil {
		return
	}
	count := int(response.Data[0])
	length := len(response.Data) - 1
	if count != length {
		err = fmt.Errorf("modbus: response data size '%v' does not match count '%v'", length, count)
		return
	}
	results = response.Data[1:]
	return
}

func TestRTU(t *testing.T) {
	ParseFlags()
	handler := NewRTUClientHandler(address)
	err := handler.Connect()
	if err != nil {
		panic(err)
	}
	defer MustClose(handler)

	//client := modbus.NewClient(handler)
	var data []byte

	//data, err = client.ReadDiscreteInputs(15, 2)
	//if err != nil {
	//	panic(err)
	//}
	//log.Println(data)
	//data, err = client.ReadWriteMultipleRegisters(0, 2, 2, 2, []byte{1, 2, 3, 4})
	//if err != nil {
	//	panic(err)
	//}
	//log.Println(data)

	data, err = ReadInputRegisters(handler, 64, 1)
	//data, err = client.ReadInputRegisters(64, 1)
	if err != nil {
		panic(err)
	}
	log.Println(data)
}
