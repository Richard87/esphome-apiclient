package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	esphome "github.com/richard87/esphome-apiclient"
	"github.com/richard87/esphome-apiclient/pb"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// ESPHomeConfig represents the subset of an ESPHome YAML config we need.
type ESPHomeConfig struct {
	ESPHome struct {
		Name string `yaml:"name"`
	} `yaml:"esphome"`
	API struct {
		Encryption struct {
			Key string `yaml:"key"`
		} `yaml:"encryption"`
	} `yaml:"api"`
	WiFi struct {
		SSID string `yaml:"ssid"`
	} `yaml:"wifi"`
}

func main() {
	var (
		yamlPath    string
		address     string
		encKey      string
		deviceName  string
		mode        string
		logLevel    string
		switchKey   uint
		switchState string
		timeout     time.Duration
	)

	flag.StringVar(&yamlPath, "yaml", "", "Path to ESPHome device YAML config file")
	flag.StringVar(&address, "address", "", "Device address (host:port). Overrides YAML-derived address.")
	flag.StringVar(&encKey, "key", "", "Base64 Noise encryption key. Overrides YAML config.")
	flag.StringVar(&deviceName, "name", "", "Expected device name (for noise validation)")
	flag.StringVar(&mode, "mode", "sensors", "Mode: sensors, logs, switch, info, entities")
	flag.StringVar(&logLevel, "log-level", "DEBUG", "Log level for logs mode (NONE, ERROR, WARN, INFO, CONFIG, DEBUG, VERBOSE, VERY_VERBOSE)")
	flag.UintVar(&switchKey, "switch-key", 0, "Switch entity key (for switch mode)")
	flag.StringVar(&switchState, "switch-state", "", "Switch state: on or off (for switch mode)")
	flag.DurationVar(&timeout, "timeout", 5*time.Second, "Connection timeout")
	flag.Parse()

	// If YAML is provided, parse it
	if yamlPath != "" {
		cfg, err := parseYAML(yamlPath)
		if err != nil {
			log.Fatalf("Failed to parse YAML: %v", err)
		}
		if deviceName == "" {
			deviceName = cfg.ESPHome.Name
		}
		if encKey == "" {
			encKey = cfg.API.Encryption.Key
		}
		// Derive address from device name if not specified
		if address == "" && deviceName != "" {
			address = deviceName + ".local:6053"
		}
	}

	if address == "" {
		fmt.Fprintln(os.Stderr, "Error: --address or --yaml is required")
		flag.Usage()
		os.Exit(1)
	}

	// Build client options
	var opts []esphome.Option
	opts = append(opts, esphome.WithClientInfo("esphome-cli"))
	if encKey != "" {
		opts = append(opts, esphome.WithEncryptionKey(encKey))
	}
	if deviceName != "" {
		opts = append(opts, esphome.WithExpectedName(deviceName))
	}
	logger := log.New(os.Stderr, "[esphome-cli] ", log.LstdFlags)
	opts = append(opts, esphome.WithLogger(logger))
	opts = append(opts, esphome.WithReconnect(5*time.Second))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Printf("Connecting to %s...", address)
	client, err := esphome.DialWithContext(ctx, address, timeout, opts...)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	logger.Printf("Connected to %s (server: %s)", client.Name(), client.ServerInfo())

	switch mode {
	case "info":
		runInfo(client)
	case "entities":
		runEntities(client)
	case "sensors":
		runSensors(ctx, client)
	case "logs":
		runLogs(ctx, client, logLevel)
	case "switch":
		runSwitch(client, uint32(switchKey), switchState)
	default:
		log.Fatalf("Unknown mode: %s", mode)
	}
}

func parseYAML(path string) (*ESPHomeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var cfg ESPHomeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	return &cfg, nil
}

func runInfo(client *esphome.Client) {
	info, err := client.DeviceInfo()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}
	fmt.Printf("Device Info:\n")
	fmt.Printf("  Name:            %s\n", info.Name)
	fmt.Printf("  MAC Address:     %s\n", info.MacAddress)
	fmt.Printf("  ESPHome Version: %s\n", info.EsphomeVersion)
	fmt.Printf("  Model:           %s\n", info.Model)
	fmt.Printf("  Project Name:    %s\n", info.ProjectName)
	fmt.Printf("  Project Version: %s\n", info.ProjectVersion)
	fmt.Printf("  Compilation Time:%s\n", info.CompilationTime)
	major, minor := client.APIVersion()
	fmt.Printf("  API Version:     %d.%d\n", major, minor)
}

func runEntities(client *esphome.Client) {
	entities, err := client.ListEntities()
	if err != nil {
		log.Fatalf("Failed to list entities: %v", err)
	}
	fmt.Printf("Found %d entities:\n\n", len(entities))

	reg := client.Entities()

	// Sensors
	sensors := reg.Sensors()
	if len(sensors) > 0 {
		fmt.Printf("Sensors (%d):\n", len(sensors))
		for _, s := range sensors {
			fmt.Printf("  [0x%08X] %s (%s) unit=%s\n", s.Key, s.Name, s.ObjectID, s.UnitOfMeasurement)
		}
		fmt.Println()
	}

	// Binary Sensors
	binarySensors := reg.BinarySensors()
	if len(binarySensors) > 0 {
		fmt.Printf("Binary Sensors (%d):\n", len(binarySensors))
		for _, s := range binarySensors {
			fmt.Printf("  [0x%08X] %s (%s)\n", s.Key, s.Name, s.ObjectID)
		}
		fmt.Println()
	}

	// Switches
	switches := reg.Switches()
	if len(switches) > 0 {
		fmt.Printf("Switches (%d):\n", len(switches))
		for _, s := range switches {
			fmt.Printf("  [0x%08X] %s (%s)\n", s.Key, s.Name, s.ObjectID)
		}
		fmt.Println()
	}

	// Lights
	lights := reg.Lights()
	if len(lights) > 0 {
		fmt.Printf("Lights (%d):\n", len(lights))
		for _, l := range lights {
			fmt.Printf("  [0x%08X] %s (%s)\n", l.Key, l.Name, l.ObjectID)
		}
		fmt.Println()
	}

	// Services
	services := client.Services().All()
	if len(services) > 0 {
		fmt.Printf("Services (%d):\n", len(services))
		for _, svc := range services {
			argNames := make([]string, 0, len(svc.Args))
			for _, a := range svc.Args {
				argNames = append(argNames, fmt.Sprintf("%s:%s", a.Name, a.Type))
			}
			fmt.Printf("  [0x%08X] %s(%s)\n", svc.Key, svc.Name, strings.Join(argNames, ", "))
		}
		fmt.Println()
	}
}

func runSensors(ctx context.Context, client *esphome.Client) {
	// First list entities to populate the registry
	if _, err := client.ListEntities(); err != nil {
		log.Fatalf("Failed to list entities: %v", err)
	}

	fmt.Println("Streaming sensor values (press Ctrl+C to stop)...")
	fmt.Println()

	_, err := client.SubscribeStates(func(msg proto.Message) {
		reg := client.Entities()
		switch m := msg.(type) {
		case *pb.SensorStateResponse:
			entity := reg.ByKey(m.Key)
			name := fmt.Sprintf("0x%08X", m.Key)
			unit := ""
			if entity != nil {
				name = entity.GetName()
				if s, ok := entity.(*esphome.SensorEntity); ok {
					unit = s.UnitOfMeasurement
				}
			}
			if m.MissingState {
				fmt.Printf("[sensor] %s = <missing>\n", name)
			} else {
				fmt.Printf("[sensor] %s = %.4g %s\n", name, m.State, unit)
			}
		case *pb.BinarySensorStateResponse:
			entity := reg.ByKey(m.Key)
			name := fmt.Sprintf("0x%08X", m.Key)
			if entity != nil {
				name = entity.GetName()
			}
			if m.MissingState {
				fmt.Printf("[binary_sensor] %s = <missing>\n", name)
			} else {
				fmt.Printf("[binary_sensor] %s = %v\n", name, m.State)
			}
		case *pb.SwitchStateResponse:
			entity := reg.ByKey(m.Key)
			name := fmt.Sprintf("0x%08X", m.Key)
			if entity != nil {
				name = entity.GetName()
			}
			fmt.Printf("[switch] %s = %v\n", name, m.State)
		case *pb.TextSensorStateResponse:
			entity := reg.ByKey(m.Key)
			name := fmt.Sprintf("0x%08X", m.Key)
			if entity != nil {
				name = entity.GetName()
			}
			if m.MissingState {
				fmt.Printf("[text_sensor] %s = <missing>\n", name)
			} else {
				fmt.Printf("[text_sensor] %s = %s\n", name, m.State)
			}
		default:
			// Print generic info for other state types
			fmt.Printf("[state] type=%T\n", msg)
		}
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to states: %v", err)
	}

	<-ctx.Done()
	fmt.Println("\nStopping...")
}

func runLogs(ctx context.Context, client *esphome.Client, levelStr string) {
	level := parseLogLevel(levelStr)

	fmt.Printf("Streaming logs (level >= %s, press Ctrl+C to stop)...\n\n", levelStr)

	_, err := client.SubscribeLogs(level, func(msg *pb.SubscribeLogsResponse) {
		levelName := logLevelName(msg.Level)
		text := string(msg.Message)
		// Remove trailing newline if present
		text = strings.TrimRight(text, "\n\r")
		fmt.Printf("[%s] %s\n", levelName, text)
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to logs: %v", err)
	}

	<-ctx.Done()
	fmt.Println("\nStopping...")
}

func runSwitch(client *esphome.Client, key uint32, stateStr string) {
	if key == 0 {
		// List available switches
		if _, err := client.ListEntities(); err != nil {
			log.Fatalf("Failed to list entities: %v", err)
		}
		switches := client.Entities().Switches()
		if len(switches) == 0 {
			fmt.Println("No switch entities found on this device.")
			return
		}

		sort.Slice(switches, func(i, j int) bool { return switches[i].Name < switches[j].Name })

		fmt.Println("Available switches:")
		for _, s := range switches {
			state := "OFF"
			if s.State {
				state = "ON"
			}
			fmt.Printf("  [0x%08X] %s = %s\n", s.Key, s.Name, state)
		}
		fmt.Println("\nUsage: --mode switch --switch-key <KEY> --switch-state on|off")
		return
	}

	var state bool
	switch strings.ToLower(stateStr) {
	case "on", "true", "1":
		state = true
	case "off", "false", "0":
		state = false
	default:
		log.Fatalf("Invalid switch state: %q (expected on/off)", stateStr)
	}

	// Populate entities for validation
	if _, err := client.ListEntities(); err != nil {
		log.Printf("Warning: could not list entities for validation: %v", err)
	}

	if err := client.SetSwitch(key, state); err != nil {
		log.Fatalf("Failed to set switch: %v", err)
	}

	stateLabel := "OFF"
	if state {
		stateLabel = "ON"
	}
	fmt.Printf("Switch 0x%08X set to %s\n", key, stateLabel)
}

func parseLogLevel(s string) pb.LogLevel {
	switch strings.ToUpper(s) {
	case "NONE":
		return pb.LogLevel_LOG_LEVEL_NONE
	case "ERROR":
		return pb.LogLevel_LOG_LEVEL_ERROR
	case "WARN", "WARNING":
		return pb.LogLevel_LOG_LEVEL_WARN
	case "INFO":
		return pb.LogLevel_LOG_LEVEL_INFO
	case "CONFIG":
		return pb.LogLevel_LOG_LEVEL_CONFIG
	case "DEBUG":
		return pb.LogLevel_LOG_LEVEL_DEBUG
	case "VERBOSE":
		return pb.LogLevel_LOG_LEVEL_VERBOSE
	case "VERY_VERBOSE":
		return pb.LogLevel_LOG_LEVEL_VERY_VERBOSE
	default:
		return pb.LogLevel_LOG_LEVEL_DEBUG
	}
}

func logLevelName(l pb.LogLevel) string {
	switch l {
	case pb.LogLevel_LOG_LEVEL_NONE:
		return "NONE"
	case pb.LogLevel_LOG_LEVEL_ERROR:
		return "ERROR"
	case pb.LogLevel_LOG_LEVEL_WARN:
		return "WARN"
	case pb.LogLevel_LOG_LEVEL_INFO:
		return "INFO"
	case pb.LogLevel_LOG_LEVEL_CONFIG:
		return "CONFIG"
	case pb.LogLevel_LOG_LEVEL_DEBUG:
		return "DEBUG"
	case pb.LogLevel_LOG_LEVEL_VERBOSE:
		return "VERBOSE"
	case pb.LogLevel_LOG_LEVEL_VERY_VERBOSE:
		return "VERY_VERBOSE"
	default:
		return fmt.Sprintf("LEVEL_%d", int(l))
	}
}
