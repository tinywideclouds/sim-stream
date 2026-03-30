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
	mqttTopic  = "zigbee2mqtt/#" // <-- We are now intercepting Z2M directly
)

func main() {
	// 1. Define what happens when raw Z2M data arrives
	var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		// Z2M topics usually look like "zigbee2mqtt/Friendly_Name"
		fmt.Printf("Source: %s\nPayload: %s\n---\n", msg.Topic(), string(msg.Payload()))
	}

	// 2. Set up connection
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID("go_z2m_direct_listener")
	opts.SetDefaultPublishHandler(messageHandler)

	// 3. Connect to the Mosquitto broker
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to Mosquitto: %v", token.Error())
	}
	fmt.Println("Connected! Intercepting raw Zigbee2MQTT data...")

	// 4. Subscribe to the Z2M firehose
	if token := client.Subscribe(mqttTopic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe: %v", token.Error())
	}

	// 5. Keep running until Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nDisconnecting...")
	client.Disconnect(250)
}
