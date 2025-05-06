package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"biathlonPrototype/internal/config"
	"biathlonPrototype/internal/processing"
	"biathlonPrototype/internal/report"
)

const (
	configFile       = "testdata\\config.json"
	eventsFile       = "testdata\\events.log"
	outputLogFile    = "results\\output.log"
	outputReportFile = "results\\final_report.txt"
)

// main serves as the entry point of the program, handling configuration loading, event processing, and report generation
func main() {
	cfg, err := config.LoadConfiguration(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Configuration loaded.")

	simulator := processing.NewSimulator(cfg)
	fmt.Println("Simulator created.")

	fmt.Printf("Loading events from %s...\n", eventsFile)
	err = simulator.LoadEventsFromFile(eventsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing events: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Event processing completed.")

	fmt.Printf("Writing log to %s...\n", outputLogFile)
	err = writeLinesToFile(outputLogFile, simulator.OutputLog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing log: %v\n", err)
	} else {
		fmt.Println("Log written.")
	}

	fmt.Println("Generating report...")
	sortedCompetitors := simulator.GetSortedCompetitors()
	reportLines := report.GenerateReport(sortedCompetitors)

	fmt.Printf("Writing report to %s...\n", outputReportFile)
	err = writeLinesToFile(outputReportFile, reportLines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Report written.")
	fmt.Println("Program completed successfully.")
}

// writeLinesToFile writes a slice of lines to a file
func writeLinesToFile(filePath string, lines []string) (err error) {
	dir := filepath.Dir(filePath)
	if mkdirErr := os.MkdirAll(dir, 0755); mkdirErr != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, mkdirErr)
	}

	var file *os.File
	file, err = os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("error closing file %s: %w", filePath, closeErr)
		} else if closeErr != nil {
			fmt.Fprintf(os.Stderr, "Additional error while closing file %s: %v (original error: %v)\n", filePath, closeErr, err)
		}
	}()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err = writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("error writing line to file %s: %w", filePath, err)
		}
	}

	if err = writer.Flush(); err != nil {
		return fmt.Errorf("error flushing buffer to file %s: %w", filePath, err)
	}
	return nil
}
