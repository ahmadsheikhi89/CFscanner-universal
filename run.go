package main

import (
	configuration "CFScanner/configuration"
	"CFScanner/scanner"
	"CFScanner/tui"
	"CFScanner/utils"
	"CFScanner/vpn"
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func run() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     os.Args[0],
		Short:   codename,
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(VersionStatement())

			if Vpn {
				utils.CreateDir(configuration.DIR)
				// Clean up old config files from previous runs
				cleanupOldConfigFiles()
			}

			utils.CreateDir(configuration.RESULTDIR)

			// set file output type
			var outputType string
			if writerType == "csv" {
				outputType = configuration.CSVInterimResultsPath
			}
			if writerType == "json" {
				outputType = configuration.JSONInterimResultsPath
			}

			if err := configuration.CreateInterimResultsFile(outputType, nTries, writerType); err != nil {
				fmt.Printf("Error creating interim results file: %v\n", err)
			}
			// number of threads for scanning
			threadsCount := threads

			// lists of ip for scanning process
			var LIST []string

			LIST = IOScanner(LIST)

			// Parsing and Validating IP LISTS
			bigIPList = utils.IPParser(LIST)

			// Shuffling IPList
			if shuffle {
				rand.Shuffle(len(bigIPList), func(i, j int) {
					bigIPList[i], bigIPList[j] = bigIPList[j], bigIPList[i]
				})
			}

			// Total number of IPS
			numberIPS := utils.TotalIps(bigIPList)

			if int(numberIPS) <= 0 {
				log.Fatal("Scanning Failed : No IP detected")
			}

			var config configuration.Configuration

			if configPath != "" {
				// Load config from file first
				config = configuration.Configuration{}.CreateTestConfig(configPath)

				// Override with explicit command line flags (preserving config file values for others)
				config.Config.FrontingTimeout = frontingTimeout
				config.Worker.Threads = threads
				config.Worker.Vpn = Vpn
				config.Worker.Download.MinDlSpeed = minDLSpeed
				config.Worker.Download.MaxDlTime = maxDLTime
				config.Worker.Download.MaxDlLatency = maxDLLatency
				config.Worker.Upload.MinUlSpeed = minULSpeed
				config.Worker.Upload.MaxUlTime = maxULTime
				config.Worker.Upload.MaxUlLatency = maxULLatency
				config.Config.TestBool.DoUploadTest = doUploadTest
				config.Config.TestBool.DoFrontingTest = fronting
				config.Shuffling = shuffle
				config.LogLevel = Loglevel

				// Handle test URL
				if testUrl != "http://google.com/generate_204" || config.Config.TestUrl == "" {
					config.Config.TestUrl = testUrl
				}

				// For nTries and writer, only override if explicitly set via command line
				// (Check if they differ from their defaults)
				if nTries != 1 { // Default for nTries is 1
					config.Config.NTries = nTries
				}
				if writerType != "csv" { // Default for writer is "csv"
					config.Config.Writer = writerType
				}
			} else {
				// No config file, use command line values
				config = configuration.Configuration{
					Config: configuration.ConfigStruct{
						FrontingTimeout: frontingTimeout,
						TestUrl:         testUrl,
						NTries:          nTries,
						Writer:          writerType,
						TestBool: configuration.TestBool{
							DoUploadTest:   doUploadTest,
							DoFrontingTest: fronting,
						},
					},

					Worker: configuration.Worker{
						Threads: threads,
						Vpn:     Vpn,
						Download: struct {
							MinDlSpeed   float64
							MaxDlTime    float64
							MaxDlLatency float64
						}{MinDlSpeed: minDLSpeed, MaxDlTime: maxDLTime, MaxDlLatency: maxDLLatency},
						Upload: struct {
							MinUlSpeed   float64
							MaxUlTime    float64
							MaxUlLatency float64
						}{MinUlSpeed: minULSpeed, MaxUlTime: maxULTime, MaxUlLatency: maxULLatency},
					},

					Shuffling: shuffle,
					LogLevel:  Loglevel,
				}
			}

			// Set environment variable early for TUI mode detection
			os.Setenv("CFSCANNER_TUI_MODE", "1")

			timer := time.Now()

			// Initialize TUI first
			tuiController := tui.NewController(int64(numberIPS), threadsCount)

			// Show version and configuration in TUI instead of stdout
			// Show version and configuration in TUI instead of stdout
			config.PrintInformationForTUI(tuiController)
			if config.Worker.Vpn {
				// Suppress Xray version output - it will be shown in TUI
				vpn.XRayVersionQuiet()
			}

			// Begin scanning process with TUI
			scanner.StartWithTUI(config, config.Worker, bigIPList, threadsCount, tuiController)

			fmt.Println("\nResults Written in :", outputType)
			fmt.Println("Sorted IPS Written in :", configuration.FinalResultsPathSorted)
			fmt.Println("Time Elapse :", time.Since(timer))
		},
	}
	return rootCmd
}

func IOScanner(LIST []string) []string {
	file, _ := utils.Exists(subnets)

	if file && subnets != "" {
		subnetFilePath := subnets
		subnetFile, err := os.Open(subnetFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer func(subnetFile *os.File) {
			err := subnetFile.Close()
			if err != nil {

			}
		}(subnetFile)

		newScanner := bufio.NewScanner(subnetFile)
		for newScanner.Scan() {
			LIST = append(LIST, strings.TrimSpace(newScanner.Text()))
		}
		if err := newScanner.Err(); err != nil {
			log.Fatal(err)
		}

	} else {
		// type conversion of string subnet to []string
		var subnet []string
		subnet = append(subnet, subnets)

		ips := utils.IPParser(subnet)

		LIST = append(LIST, ips...)

	}
	return LIST
}

// cleanupOldConfigFiles removes old xray config files from previous runs
func cleanupOldConfigFiles() {
	configDir := configuration.DIR

	// Read directory
	files, err := os.ReadDir(configDir)
	if err != nil {
		return // Directory doesn't exist or can't be read
	}

	// Remove old config files
	for _, file := range files {
		fileName := file.Name()
		// Remove old per-IP config files and temporary xray files
		if (strings.HasPrefix(fileName, "config-") || strings.HasPrefix(fileName, "xray-")) && strings.HasSuffix(fileName, ".json") {
			filePath := fmt.Sprintf("%s/%s", configDir, fileName)
			os.Remove(filePath) // Ignore errors
		}
	}
}
