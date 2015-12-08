# onion

[![GoDoc](https://godoc.org/github.com/crackcomm/onion?status.svg)](https://godoc.org/github.com/crackcomm/onion)

Make an onion made of net and crypto layers.

```Go
o := onion.New(
  net.NewLayer(),
  tor.NewLayer(
    tor.WithPort(80), // Hidden service port
    tor.WithBin("/usr/bin/tor"),
    tor.WithVerbose(true),
  ),
  sch.NewLayer(
    sch.WithPubKey(readPubKey()),
    sch.WithPrivKey(readPrivKey()),
  ),
  tls.NewLayer(
    tls.WithCertKeyFile("ca.pem", "ca.key"),
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
```

## TOR Hidden service

To create a tor hidden service, all You need to do is create an Onion:

```Go
o := onion.New(
  net.NewLayer(),
  tor.NewLayer(
    tor.WithPort(80), // Hidden service port
    tor.WithBin("/usr/bin/tor"),
  ),
)
```

Then you can start listening through TOR:

```Go
listener, err := o.Listener(nil)
if err != nil {
	glog.Fatal(err)
}

// Will output {address}.onion
glog.Infof("Listening on %s", listener.Addr())

for {
	c, err := listener.Accept()
	if err != nil {
		glog.Warning(err)
		continue
	}

	go serve(c)
}
```

You can also dial to TOR `.onion` services using this Onion:

```Go
conn, err := o.Connect("{address}.onion", time.Minute)
if err != nil {
	glog.Fatal(err)
}

conn.Write([]byte("Hello world!\n"))
```
