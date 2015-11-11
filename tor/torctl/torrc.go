package torctl

import (
	"bytes"
	"text/template"
)

// TorrcBody - Generates a torrc configuration file body from launch options.
func TorrcBody(opts *LaunchOptions) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := TorrcTemplate.Execute(buf, opts)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// TorrcTemplate - Default torrc config file template.
var TorrcTemplate *template.Template
var templateString, _ = torrcBytes()

func init() {
	TorrcTemplate = template.Must(template.New("").Parse(string(templateString)))
}
