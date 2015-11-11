# onion

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
