package utils

import (
	"fmt"
	"github.com/streadway/amqp"
	"os/exec"
	"time"
)


func Ping(ip string) bool {
	run := exec.Command("Ping", ip, "-c 1", "-w 1")
	err := run.Run()
	return err == nil
}


type RemoteLogger struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

func (r *RemoteLogger) Connect(connString string) error {
	var err error
	r.conn, err = amqp.Dial(connString)
	if err != nil {
		return err
	}
	r.channel, err = r.conn.Channel()
	if err != nil {
		return err
	}
	err = r.channel.ExchangeDeclare(
		"logging",
		"topic",
		false,
		false,
		false,
		false,
		nil,
	)
	return err
}

func (r *RemoteLogger) Close() {
	r.channel.Close()
	r.conn.Close()
}

func (r *RemoteLogger) Write(msg []byte) (int, error) {
	datetime := time.Now().Format(time.UnixDate)
	sendmsg := fmt.Sprintf("%s PINGER %v", datetime, string(msg))
	err := r.channel.Publish(
		"logging",
		"ping",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(sendmsg),
		})
	written := len(msg)
	if err != nil {
		written = 0
	}
	return written, err
}

func NewRemoteLogger(connString string) (*RemoteLogger, error) {
	var r RemoteLogger
	err := r.Connect(connString)
	r.exchange = "logging"
	return &r, err
}