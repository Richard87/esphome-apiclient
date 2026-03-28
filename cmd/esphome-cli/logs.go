package main

import (
	"context"
	"fmt"
	"strings"

	esphome "github.com/richard87/esphome-apiclient"
	"github.com/richard87/esphome-apiclient/pb"
)

func runLogs(ctx context.Context, client *esphome.Client, levelStr string) error {
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
		return fmt.Errorf("failed to subscribe to logs: %w", err)
	}

	<-ctx.Done()
	fmt.Println("\nStopping...")
	return nil
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
