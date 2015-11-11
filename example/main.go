package main

import (
	"bufio"
	"flag"
	"io"
	"net"
	"os"
	"time"

	"github.com/golang/glog"

	"github.com/crackcomm/onion"
	netl "github.com/crackcomm/onion/net"
	"github.com/crackcomm/onion/sch"
	"github.com/crackcomm/onion/tls"
	"github.com/crackcomm/onion/tor"
)

var (
	client   = flag.Bool("client", false, "client")
	torProxy = flag.String("proxy", "", "proxy address (socks5)")
	torBin   = flag.String("tor-bin", "/usr/local/bin/tor", "Tor binary")
	pubKey   = flag.String("pub-key", "root.pub", "public key")
	privKey  = flag.String("priv-key", "root.key", "private key")
	verbose  = flag.Bool("verbose", false, "verbose tor")
)

func main() {
	defer glog.Flush()
	flag.Parse()

	if *torBin == "" {
		glog.Fatal("Tor-bin empty")
	}

	glog.Info("start")

	// Create an onion
	o := onion.New(
		netl.NewLayer(),
		tor.NewLayer(
			tor.WithBin(*torBin),
			tor.WithPort(80),
			tor.WithVerbose(*verbose),
			tor.WithProxy(*torProxy),
		),
		sch.NewLayer(
			sch.WithPubKey(readPubKey()),
			sch.WithPrivKey(readPrivKey()),
		),
		tls.NewLayer(
			tls.WithInsecure(),
			tls.WithCertAndKeyFile("ca.pem", "ca.key"),
		),
		sch.NewLayer(
			sch.WithPubKey(readPubKey()),
			sch.WithPrivKey(readPrivKey()),
		),
		sch.NewLayer(
			sch.WithPubKey(readPubKey()),
			sch.WithPrivKey(readPrivKey()),
		),
	)
	defer o.Close()

	if *client {
		if flag.Arg(0) == "" {
			glog.Info("Usage: example -client {address}")
			return
		}
		conn, err := o.Connect(flag.Arg(0), time.Minute)
		if err != nil {
			glog.Fatal(err)
		}
		conn.Write([]byte("Hello world!\n"))

		reader := bufio.NewReader(conn)
		line, err := reader.ReadSlice('\n')
		if err != nil {
			glog.Warning(err)
			return
		}

		glog.Infof("%s", line)
		return

	}

	listener, err := o.Listener(nil)
	if err != nil {
		glog.Fatal(err)
	}

	glog.Infof("Listening on %s", listener.Addr())

	for {
		c, err := listener.Accept()
		if err != nil {
			glog.Warning(err)
			continue
		}

		go serve(c)
	}
}

func serve(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadSlice('\n')
		if err != nil {
			glog.Warning(err)
			return
		}

		glog.Infof("%s", line)
	}

}

func readPubKey() (key *[32]byte) {
	f, err := os.Open(*pubKey)
	if err != nil {
		glog.Fatal(err)
	}
	key = new([32]byte)
	_, err = io.ReadFull(f, key[:])
	if err != nil {
		glog.Fatal(err)
	}
	return
}

func readPrivKey() (key *[64]byte) {
	f, err := os.Open(*privKey)
	if err != nil {
		glog.Fatal(err)
	}
	key = new([64]byte)
	_, err = io.ReadFull(f, key[:])
	if err != nil {
		glog.Fatal(err)
	}
	return
}
