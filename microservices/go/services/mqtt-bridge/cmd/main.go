package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	mqttBroker = "tcp://127.0.0.1:1883"
	mqttTopic  = "homeassistant/#" // The wildcard '#' grabs everything under 'homeassistant'
)

func main() {
	// 1. Define what happens when a message arrives
	var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Topic: %s\nPayload: %s\n---\n", msg.Topic(), string(msg.Payload()))
	}

	// 2. Set up the connection options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID("go_local_logger")
	opts.SetDefaultPublishHandler(messageHandler)

	// 3. Connect to Mosquitto
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to Mosquitto: %v", token.Error())
	}
	fmt.Println("Connected to local Mosquitto broker!")

	// 4. Subscribe to the stream
	if token := client.Subscribe(mqttTopic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe: %v", token.Error())
	}
	fmt.Printf("Subscribed to: %s\nWaiting for Home Assistant data...\n---\n", mqttTopic)

	// 5. Keep the program running until you press Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nDisconnecting...")
	client.Disconnect(250)
}
