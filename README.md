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
  nacl.NewLayer(
    nacl.WithPubKey(readPubKey()),
    nacl.WithPrivKey(readPrivKey()),
  ),
  tls.NewLayer(
    tls.WithCertKeyFile("ca.pem", "ca.key"),
  ),
  nacl.NewLayer(
    nacl.WithPubKey(readPubKey()),
    nacl.WithPrivKey(readPrivKey()),
  ),
  nacl.NewLayer(
    nacl.WithPubKey(readPubKey()),
    nacl.WithPrivKey(readPrivKey()),
  ),
)
```
