package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"OPiControllerPinger/utils"
	"github.com/streadway/amqp"
)

const rabbitMqConnection = "amqp://guest:guest@localhost:5672/"

type address struct {
	ip         string
	state      bool
	lastChange time.Time
}

func (a *address) StrFormat() string {
	return fmt.Sprintf("IP: %s, online: %t since: %v", a.ip, a.state, a.lastChange)
}

func (a *address) MarshalJSON() ([]byte, error) {
	ip, _ := json.Marshal(a.ip)
	state, _ := json.Marshal(a.state)
	lastChange, _ := json.Marshal(a.lastChange)
	return []byte(fmt.Sprintf("[%v, %v, %v]", string(ip), string(state), string(lastChange))), nil
}

func NewAddress(ip string) *address {
	return &address{
		ip:         ip,
		state:      false,
		lastChange: time.Now(),
	}
}

func readSettings(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening file: %v\n", filename)
	}
	defer file.Close()
	var addressList []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := scanner.Text()
		if len(ip) > 6 {
			addressList = append(addressList, ip)
		}
		if err := scanner.Err(); err != nil {
			log.Fatal("Error reading file")
		}
	}
	return addressList
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func openLogFile() *os.File {
	fileObj, err := os.OpenFile("log.txt", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	failOnError(err, "Error opening logfile")
	return fileObj
}

func updateState(addressList []*address) error {
	for idx, addr := range addressList {
		prevstate := addr.state
		addr.state = utils.Ping(addr.ip)
		if addr.state != prevstate {
			addr.lastChange = time.Now()
		}
		addressList[idx] = addr
	}
	return nil
}

func main() {
	const settingsFile = "settings.ini"

	logfile := openLogFile()
	defer logfile.Close()
	remoteLogger, err := utils.NewRemoteLogger(rabbitMqConnection)
	defer remoteLogger.Close()
	if err != nil {
		os.Exit(1)
	}
	logWriter := io.MultiWriter(logfile, remoteLogger)
	log.SetOutput(logWriter)
	ipList := readSettings(settingsFile)
	var addressList []*address
	for _, address := range ipList {
		addressList = append(addressList, NewAddress(address))
	}
	err = updateState(addressList)
	conn, err := amqp.Dial(rabbitMqConnection)
	failOnError(err, "Failed to connect Rabbit")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"messages",
		"topic",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare an exchange")
	for {
		msg, err := json.Marshal(addressList)
		failOnError(err, "Failed to JSON-ify message")
		err = ch.Publish(
			"messages",
			"Ping",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        msg,
			})
		log.Println(string(msg))
		failOnError(err, "Failed to publish message")
		time.Sleep(5 * time.Second)
	}
}
