package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

//const PARENB = 0000400
//const CSTOPB = 0000100
//const CSIZE = 0000060
//const CS8 = 0000060
const CRTSCTS = 020000000000
//const TCSANOW = 0
const SEEK_SET int64 = 0

const PORT0_PIN_NUM uint = 6
const PORT1_PIN_NUM uint = 10

const PORT0_NAME = "/dev/ttyAMA2"
const PORT1_NAME = "/dev/ttyUSB0"

const NPORTS = 2
const BAUDRATE uint32 = 0000016

var fdPorts [NPORTS]int
var dePins [NPORTS]*os.File
var portsName = [NPORTS]string{
	PORT0_NAME,
	PORT1_NAME,
}

func SyscallClose(fd int) {
	err := syscall.Close(fd)
	log.Printf("SyscallClose %d %v", fd, err)
}

func CloseFile(f io.Closer) {
	err := f.Close()
	log.Printf("CloseFile %v", err)
}

func main() {
	var err error
	var fdPort0, fdPort1 int
	log.Printf("PORT0_PIN_NUM %d\n", PORT0_PIN_NUM)
	fdDe0 := configureIopin(PORT0_PIN_NUM, "out", true)
	defer unconfigureIopin(PORT0_PIN_NUM)
	defer CloseFile(fdDe0)
	log.Printf("PORT1_PIN_NUM %d\n", PORT1_PIN_NUM)
	fdDe1 := configureIopin(PORT1_PIN_NUM, "out", true)
	defer unconfigureIopin(PORT1_PIN_NUM)
	defer CloseFile(fdDe1)
	fdPort0, err = syscall.Open(PORT0_NAME, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NDELAY/*O_SYNC*/, 0600)
	if err != nil {
		panic(err)
	}
	defer SyscallClose(fdPort0)
	fdPort1, err = syscall.Open(PORT1_NAME, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NDELAY/*O_SYNC*/, 0600)
	if err != nil {
		panic(err)
	}
	defer SyscallClose(fdPort1)
	fdPorts[0] = fdPort0
	fdPorts[1] = fdPort1
	dePins[0] = fdDe0
	dePins[1] = fdDe1

	for iport := 0; iport < NPORTS; iport++ {
		portSettings := &syscall.Termios{}
		fd := fdPorts[iport]
		err = tcgetattr(fd, portSettings)
		if err != nil {
			// Warning only.
			panic(err)
		}
		cfSetIspeed(portSettings, BAUDRATE)

		portSettings.Cflag &= ^uint32(syscall.PARENB)
		log.Println("Cflag", portSettings.Cflag)
		portSettings.Cflag &= ^uint32(syscall.CSTOPB)
		log.Println("Cflag", portSettings.Cflag)
		portSettings.Cflag &= ^uint32(syscall.CSIZE)
		log.Println("Cflag", portSettings.Cflag)
		portSettings.Cflag |= syscall.CS8
		log.Println("Cflag", portSettings.Cflag)
		portSettings.Cflag &= ^uint32(CRTSCTS)
		log.Println("Cflag", portSettings.Cflag)
		portSettings.Cflag |= syscall.CREAD | syscall.CLOCAL
		log.Println("Cflag", portSettings.Cflag)
		portSettings.Iflag &= ^uint32(syscall.IGNBRK|syscall.BRKINT|syscall.PARMRK|syscall.ISTRIP|syscall.INLCR|syscall.IGNCR|syscall.ICRNL|syscall.IXON)
		log.Println("Iflag", portSettings.Iflag)
		portSettings.Lflag &= ^uint32(syscall.ECHO|syscall.ECHONL|syscall.ICANON|syscall.ISIG|syscall.IEXTEN)
		log.Println("Lflag", portSettings.Lflag)
		portSettings.Oflag &= ^uint32(syscall.OPOST)
		log.Println("Oflag", portSettings.Oflag)
		portSettings.Cc[syscall.VMIN] = 1
		log.Println("portSettings.Cc[syscall.VMIN]", syscall.VMIN, portSettings.Cc[syscall.VMIN])
		portSettings.Cc[syscall.VTIME] = 5*2
		log.Println("portSettings.Cc[syscall.VTIME]", syscall.VTIME, portSettings.Cc[syscall.VTIME])
		err = tcsetattr(fd, portSettings)
		if err != nil {
			log.Printf("can`t setting attributes for %s\n", portsName[iport])
			panic(err)
		}
	}

	var n int = 0
	var ctr uint = 0
	wbuf := make([]byte, 64)
	rbuf := make([]byte, 64)
	for i := 0; i < 100; i++ {
		senderIdx := ctr % NPORTS
		ctr++
		receiverIdx := ctr % NPORTS
		fdSender := fdPorts[senderIdx]
		fdReceiver := fdPorts[receiverIdx]
		deSender := dePins[senderIdx]
		deReceiver := dePins[receiverIdx]
		senderName := portsName[senderIdx]
		receiverName := portsName[receiverIdx]

		setPinValue(deSender, true)
		setPinValue(deReceiver, false)

		time.Sleep(time.Second)

		_, err = rand.Read(wbuf)
		if err != nil {
			panic(err)
		}

		log.Printf("send data to %s:\n", senderName)
		//log.Println(wbuf)
		n, err = syscall.Write(fdSender, wbuf)
		if err != nil {
			panic(err)
		}

		log.Printf("write %d bytes to %s ok\n", n, senderName)

		time.Sleep(time.Second)

		n, err = syscall.Read(fdReceiver, rbuf)
		if err != nil {
			panic(err)
		}

		log.Printf("read %d bytes from %s:\n", n, receiverName)

		time.Sleep(time.Second)
	}
}

func tcgetattr(fd int, termios *syscall.Termios) (err error) {
	r, _, errno := syscall.Syscall(uintptr(syscall.SYS_IOCTL),
		uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(termios)))
	if errno != 0 {
		err = errno
		return
	}
	if r != 0 {
		err = fmt.Errorf("tcgetattr failed %v", r)
		return
	}
	return
}

func tcsetattr(fd int, termios *syscall.Termios) (err error) {
	r, _, errno := syscall.Syscall(uintptr(syscall.SYS_IOCTL),
		uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(termios)))
	if errno != 0 {
		err = errno
		return
	}
	if r != 0 {
		err = fmt.Errorf("tcsetattr failed %v", r)
	}
	return
}

func cfSetIspeed(termios *syscall.Termios, speed uint32) {
	termios.Ispeed = speed
}

func configureIopin(pinNum uint, direction string, writable bool) (fd *os.File) {
	var err error
	pinName := fmt.Sprintf("%d", pinNum)
	err = ioutil.WriteFile("/sys/class/gpio/export", []byte(pinName), 0644)
	if err != nil {
		panic(err)
	}
	pinName = fmt.Sprintf("/sys/class/gpio/gpio%d/direction", pinNum)
	err = ioutil.WriteFile(pinName, []byte(direction), 0644)
	pinName = fmt.Sprintf("/sys/class/gpio/gpio%d/value", pinNum)
	flag := os.O_RDONLY
	if writable {
		flag = os.O_RDWR
	}
	fd, err = os.OpenFile(pinName, flag, 0600)
	if err != nil {
		log.Printf("pin%d:can`t open /sys/class/gpio/gpio%d/value\n", pinNum, pinNum)
		panic(err)
	}
	return fd
}

func unconfigureIopin(pinNum uint) {
	pinName := fmt.Sprintf("%d", pinNum)
	err := ioutil.WriteFile("/sys/class/gpio/unexport", []byte(pinName), 0644)
	log.Printf("unconfigureIopin %d %s %v", pinNum, pinName, err)
}

func setPinValue(fd *os.File, on bool) {
	var err error
	var data []byte
	data = []byte("0")
	if on {
		data = []byte("1")
	}
	_, err = fd.Seek(SEEK_SET, 0)
	if err != nil {
		panic(err)
	}
	_, err = syscall.Write(int(fd.Fd()), data)
	if err != nil {
		panic(err)
	}
}
