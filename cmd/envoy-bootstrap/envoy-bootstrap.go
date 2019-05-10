package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
)

var (
	envoyBinary       string
	bootstrapTemplate string
	bootstrapConfig   string
	logLevel          string
	discoveryAddress  string
	discoveryPort     string
	serviceCluster    string

	base64Encoded bool
)

type BootstrapConfig struct {
	NodeId         string
	ServiceCluster string
	Xds            *XdsConfig
}

type XdsConfig struct {
	DiscoveryType     string
	ConnectionTimeout string
	Address           string
	Port              string
}

func main() {
	flag.Parse()

	cfg := &BootstrapConfig{
		NodeId:         buildNodeId(),
		ServiceCluster: serviceCluster,
		Xds: &XdsConfig{
			DiscoveryType:     "STATIC",
			ConnectionTimeout: "5s",
			Address:           discoveryAddress,
			Port:              discoveryPort,
		},
	}
	t := template.Must(template.ParseFiles(bootstrapTemplate))

	fout, err := os.Create(bootstrapConfig)
	check(err)

	t.Execute(fout, cfg)

	cmd := exec.Command(envoyBinary, buildArgs(bootstrapConfig, logLevel)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	check(err)

	done := make(chan bool)

	go func() {
		err := cmd.Wait()
		check(err)
		done <- true
	}()

	<-done
}

func buildArgs(configPath, logLevel string) []string {
	return []string{
		"--v2-config-only",
		"-l",
		logLevel,
		"-c",
		configPath,
	}
}

func buildNodeId() string {
	return fmt.Sprintf("%s@%s", os.Getenv("POD_IP"), os.Getenv("POD_NAME"))
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&envoyBinary, "envoyBinary", "", "Input file path. Uses standard input if not provided")
	flag.StringVar(&logLevel, "logLevel", "info", "Input file path. Uses standard input if not provided")
	flag.StringVar(&bootstrapTemplate, "bootstrapTemplate", "", "Input file path. Uses standard input if not provided")
	flag.StringVar(&bootstrapConfig, "bootstrapConfig", "", "Input file path. Uses standard input if not provided")
	flag.StringVar(&discoveryAddress, "discoveryAddress", "", "Input file path. Uses standard input if not provided")
	flag.StringVar(&discoveryPort, "discoveryPort", "", "Input file path. Uses standard input if not provided")
	flag.StringVar(&serviceCluster, "serviceCluster", "", "Input file path. Uses standard input if not provided")

	flag.BoolVar(&base64Encoded, "base64", false, "Use base64 encoding to write output (encrypt mode), and base64 decoding to read input (decrypt mode)")

}
