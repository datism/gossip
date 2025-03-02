package main

import (
	"flag"
	"net"
	"os"
	"time"

	"gossip/core"
	"gossip/message"
	"gossip/transport"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:5060", "Local IP")
	flag.Parse()

	// Open a file for logging
	// logFile, err := os.OpenFile("gossip.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Error opening log file")
	// 	os.Exit(1)
	// }
	// defer logFile.Close()

	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	// Rotate log file by size using lumberjack
	log.Logger = log.Output(zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339},
		// &lumberjack.Logger{
		// 	Filename:   "gossip.log",
		// 	MaxSize:    10,   // Max size in MB
		// 	MaxBackups: 3,    // Max number of old log files to keep
		// 	MaxAge:     28,   // Max number of days to keep old log files
		// 	Compress:   true, // Compress old log files
		// },
	))

	// Create UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Error().Err(err).Msg("Error resolving UDP address")
		os.Exit(1)
	}

	// Create UDP socket
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Error().Err(err).Msg("Error creating UDP socket")
		os.Exit(1)
	}
	defer conn.Close()

	log.Info().Msgf("Listening on %s", *addr)

	// Buffer to store incoming data
	buffer := make([]byte, 1024) // 1 KB buffer

	for {
		// Read from UDP socket
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Error().Err(err).Msg("Error reading from UDP socket")
			continue
		}

		// Copy the data to prevent data races
		data := make([]byte, n)
		copy(data, buffer[:n])

		// Handle each message in a new goroutine
		go handleMessage(conn, clientAddr, data)
	}
}

func handleMessage(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte) {
	// Log received message
	log.Debug().
		Int("bytes", len(data)).
		Str("client", clientAddr.String()).
		Msg("Received message")

	msg, err := message.Parse(data)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing SIP request")
		return
	}

	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		log.Error().Msg("Error asserting local address to UDPAddr")
	}

	transport := &transport.Transport{
		Protocol:   "UDP",
		Conn:       conn,
		LocalAddr:  udpAddr,
		RemoteAddr: clientAddr,
	}
	msg.Transport = transport
	core.HandleMessage(msg)
}
