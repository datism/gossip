package proxy

import (
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/arl/statsviz"
	"github.com/datism/sip"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	_ "net/http/pprof"
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

	zerolog.SetGlobalLevel(zerolog.Disabled)

	// Rotate log file by size using lumberjack
	log.Logger = log.Output(zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339},
	// 	&lumberjack.Logger{
	// 		Filename:   "gossip.log",
	// 		MaxSize:    1000,  // Max size in MB
	// 		MaxBackups: 3,     // Max number of old log files to keep
	// 		MaxAge:     28,    // Max number of days to keep old log files
	// 		Compress:   false, // Compress old log files
	// },
	))

	go httpServer(":8080")

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

	msg, err := sip.ParseSipMessage(data, sip.ParseOptions{
		ParseFrom:       true,
		ParseTo:         true,
		ParseCseqByType: true,
		ParseTopMostVia: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("Error parsing SIP request")
		return
	}

	transport := &sip.SIPTransport{
		Protocol:   "udp",
		Conn:       conn,
		LocalAddr:  conn.LocalAddr().String(),
		RemoteAddr: clientAddr.String(),
	}
	HandleMessage(msg, transport)
}

func httpServer(address string) {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Alive"))
	})

	http.HandleFunc("/mem", func(w http.ResponseWriter, r *http.Request) {
		runtime.GC()
		stats := &runtime.MemStats{}
		runtime.ReadMemStats(stats)
		data, _ := json.MarshalIndent(stats, "", "  ")
		w.WriteHeader(200)
		w.Write(data)
	})

	http.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)

	statsviz.Register(http.DefaultServeMux)

	log.Info().Msgf("Http server started address=%s", address)
	http.ListenAndServe(address, nil)
}
