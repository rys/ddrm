//go:build client
// +build client

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"
)

// Type to describe the JSON app config on disk
type DdrmAppConfig struct {
	EmailUser           string `json:"email_user"`
	EmailUserName       string `json:"email_user_name"`
	EmailPassword       string `json:"email_password"`
	EmailServerHostname string `json:"email_server_hostname"`
	EmailServerPort     string `json:"email_server_port"`
	EmailTo             string `json:"email_to"`
	EmailToName         string `json:"email_to_name"`
	DnsServer1          string `json:"dns_server_1"`
	DnsServer2          string `json:"dns_server_2"`
	EmailSenderName     string `json:"email_sender_name"`
	EmailLink           string `json:"email_link"`
	EmailLogo           string `json:"email_logo"`
	EmailSubject        string `json:"email_subject"`
	RedisDatabase       int    `json:"redis_database"`
	RedisPassword       string `json:"redis_password"`
	RedisServer         string `json:"redis_server"`
	RedisKeyPrefix      string `json:"redis_key_prefix"`
}

// Type to describe the record checking JSON config on disk
type DdrmRecordConfig struct {
	FQDN           string         `json:"fqdn"`
	Type           DdrmRecordType `json:"type"`
	ExpectedValues []string       `json:"expected_values"`
}

// State constants
const (
	ddrmConfigFilePath        string = "ddrm.conf"
	ddrmRecordsConfigFilePath string = "ddrm-records.conf"
	ddrmEmailTemplatePath     string = "ddrm-email-template.txt"
	ddrmStartupBanner         string = "DNS Spy Record Monitor %s %s (git %s) built by %s"
	ddrmConfigPathBanner      string = "config path: %s"
	ddrmRecordsPathBanner     string = "records config path: %s"
	ddrmDebugMode             string = "debug mode enabled"
)

// Error messages
const (
	ddrmErrorNoConfigPath          string = "no configuration file at path: %s"
	ddrmErrorNoRecordsPath         string = "no records configuration file at path %s"
	ddrmErrorUnableToReadFile      string = "unable to read file: %s %#v"
	ddrmErrorUnableToStatFile      string = "unable to stat file: %s %#v"
	ddrmErrorInsecureConfig        string = "insecure config: %s %#v"
	ddrmErrorSendingMail           string = "unable to send email"
	ddrmErrorUnableToGenerateEmail string = "unable to generate email report to send"
	ddrmErrorUnableToUnmarshalJSON string = "unable to unmarshal JSON from %s %#v"
	ddrmErrorUnableToRunLoop       string = "unable to process main run loop"
	ddrmErrorUnableToFetchRecord   string = "unable to fetch record data for %s %s"
)

// Debug messages
const (
	ddrmDebugStartProcessing    string = "  processing records for: %s %s"
	ddrmDebugRetrievedDataFrom  string = "fetched record data from: %s"
	ddrmDebugEndProcessing      string = "   end of processing for: %s %s"
	ddrmDebugTryingCache        string = "        trying cache for: %s %s"
	ddrmDebugNoCache            string = "    no cached values for: %s %s"
	ddrmDebugUsingStartupConfig string = "using startup config for: %s %s"
)

// Success and reporting messages
const (
	ddrmSuccessSentMail     string = "sent email report successfully"
	ddrmReportReadingConfig string = "reading config: %s"
	ddrmReportReadConfig    string = "successfully read config: %s"
	ddrmReportReadRecords   string = "read %d record(s) to process"
	ddrmReportDNSClientErr  string = "error asking %s for %s %s record: %#v"
	ddrmSuccessSetupCronJob string = "setup periodic processing for %s (%s): %s"
)

// os.Exit() return codes to indicate exit state on error
const (
	ddrmExitOK = iota
	ddrmExitDuringConfig
	ddrmExitAfterMailTest
	ddrmExitAfterDNSClientTest
	ddrmExitErrorRunningTUI
	ddrmExitErrorCreatingScheduler
	ddrmExitErrorCreatingUIUpdateJob
	ddrmExitErrorCreatingRecordProcessorJob
)

// Application runtime state
var (
	stateDebug                 bool          = false
	stateSimplerLogging        bool          = false
	stateInsecureConfig        bool          = false
	statePlainTextEmail        bool          = false
	stateSendTestEmail         bool          = false
	stateIPV4                  bool          = true
	stateIPV6                  bool          = false
	stateTCP                   bool          = false
	stateSleep                 time.Duration = 60 * time.Second
	stateDNSClientTest         bool          = false
	stateDNSTimeout            time.Duration = 2 * time.Second
	stateConfigFilePath        string        = ddrmConfigFilePath
	stateRecordsConfigFilePath string        = ddrmRecordsConfigFilePath
	stateAllowImpreciseMatch   bool          = false
	stateExpand                bool          = false
	stateTabsToSpaces          int           = 4
	stateUseRedis              bool          = false
	stateUI                    bool          = false
	stateLogRecordProcessing   bool          = false
	stateUITickRate            time.Duration = 1 * time.Second
	stateAltScreenMode         bool          = false
	statePrintVersion          bool          = false
)

// unmarshalled application config
var ddrmAppConfig DdrmAppConfig

// unmarshalled record configs
var ddrmRecordConfig []DdrmRecordConfig

// logger and scheduler objects
var (
	ddrmLog       zerolog.Logger
	cronScheduler gocron.Scheduler
)

// Defensively try and read a config file and return the raw bytes
// Full os.Exit() if it fails
func readFileReturningBytes(filePath string) []byte {

	var config []byte

	if filePath != "" {
		// check if the file exists and we can stat it
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			dbgf(ddrmErrorUnableToReadFile, filePath, err)
			os.Exit(ddrmExitDuringConfig)
		}

		stat, err := os.Lstat(filePath)

		if err != nil {
			dbgf(ddrmErrorUnableToStatFile, filePath, err)
			os.Exit(ddrmExitDuringConfig)
		}

		if stat.Mode() != 0400 {
			if !stateInsecureConfig {
				dbgf(ddrmErrorInsecureConfig, filePath, stat.Mode().Perm().String())
				os.Exit(ddrmExitDuringConfig)
			}
		}

		dbgf(ddrmReportReadingConfig, filePath)

		config, err = os.ReadFile(filePath)
		if err != nil {
			dbgf(ddrmErrorUnableToReadFile, filePath, err)
			os.Exit(ddrmExitDuringConfig)
		}

	} else {
		dbgf(ddrmErrorNoConfigPath, filePath)
		os.Exit(ddrmExitDuringConfig)
	}

	return config
}

func readAppConfig() {
	if stateConfigFilePath != "" {

		config := readFileReturningBytes(stateConfigFilePath)

		err := json.Unmarshal(config, &ddrmAppConfig)

		if err != nil {
			dbgf(ddrmErrorUnableToUnmarshalJSON, stateConfigFilePath, err)
			os.Exit(ddrmExitDuringConfig)
		}

		dbgf(ddrmReportReadConfig, stateConfigFilePath)

	} else {
		dbgf(ddrmErrorNoConfigPath, stateConfigFilePath)
		os.Exit(ddrmExitDuringConfig)
	}
}

func readRecordsConfig() {
	ddrmRecordStates = make(map[string]DdrmRecordState, 0)

	if stateRecordsConfigFilePath != "" {

		config := readFileReturningBytes(stateRecordsConfigFilePath)

		err := json.Unmarshal(config, &ddrmRecordConfig)

		if err != nil {
			dbgf(ddrmErrorUnableToUnmarshalJSON, stateRecordsConfigFilePath, err)
			os.Exit(ddrmExitDuringConfig)
		}

		dbgf(ddrmReportReadRecords, len(ddrmRecordConfig))
		dbgf(ddrmReportReadConfig, stateRecordsConfigFilePath)

		// Re-initialise the map to hold fresh state now we know how big it should be
		ddrmRecordStates := map[string]DdrmRecordState{}
		for _, record := range ddrmRecordConfig {
			ddrmRecordStates[record.FQDN+":"+string(record.Type)] = DdrmRecordState{}
		}

	} else {
		dbgf(ddrmErrorNoConfigPath, stateRecordsConfigFilePath)
		os.Exit(ddrmExitDuringConfig)
	}
}

func setupLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	logger := zerolog.ConsoleWriter{Out: os.Stdout}

	if stateUI {
		// log to stderr instead in UI mode, without colours
		logger = zerolog.ConsoleWriter{Out: os.Stderr}
		logger.NoColor = true
	}

	// remove some bits of the output so that it looks nicer
	if stateSimplerLogging {
		logger.FormatLevel = func(i interface{}) string {
			return ""
		}

		logger.FormatTimestamp = func(i interface{}) string {
			return ""
		}
	}

	ddrmLog = zerolog.New(logger).With().Timestamp().Logger()
}

func setupPeriodicTasks() {
	cron, err := gocron.NewScheduler()

	if err != nil {
		os.Exit(ddrmExitErrorCreatingScheduler)
	}

	job, err := cron.NewJob(
		gocron.DurationJob(stateUITickRate),
		gocron.NewTask(sendUpdateUIMsg),
	)

	dbgf(ddrmSuccessSetupCronJob, "updating UI", stateUITickRate.String(), job.ID().String())

	if err != nil {
		os.Exit(ddrmExitErrorCreatingUIUpdateJob)
	}

	job, err = cron.NewJob(
		gocron.DurationJob(stateSleep),
		gocron.NewTask(processRecords),
	)

	dbgf(ddrmSuccessSetupCronJob, "processing records", stateSleep.String(), job.ID().String())

	if err != nil {
		os.Exit(ddrmExitErrorCreatingRecordProcessorJob)
	}

	cronScheduler = cron
	cronScheduler.Start()
}

func parseCliFlags() {
	flag.BoolVar(&stateDebug, "debug", stateDebug, "Enabled debug mode")
	flag.BoolVar(&stateSimplerLogging, "quieter", stateSimplerLogging, "Be quieter on output")
	flag.BoolVar(&stateInsecureConfig, "insecure", stateInsecureConfig, "Allow an insecure config")
	flag.BoolVar(&statePlainTextEmail, "plaintext", statePlainTextEmail, "Send plaintext email")
	flag.BoolVar(&stateSendTestEmail, "testemail", stateSendTestEmail, "Send a test email and exit(2)")
	flag.BoolVar(&stateIPV4, "4", stateIPV4, "Use IPv4 for DNS resolution")
	flag.BoolVar(&stateIPV6, "6", stateIPV6, "Use IPv6 for DNS resolution")
	flag.BoolVar(&stateTCP, "tcp", stateTCP, "Use TCP instead of UDP for DNS resolution")
	flag.DurationVar(&stateSleep, "sleep", stateSleep, "Seconds to sleep between checks")
	flag.DurationVar(&stateDNSTimeout, "timeout", stateDNSTimeout, "Seconds to wait for DNS client reponses before moving on")
	flag.BoolVar(&stateDNSClientTest, "testdns", stateDNSClientTest, "Test the DNS client and exit(3)")
	flag.StringVar(&stateConfigFilePath, "config", ddrmConfigFilePath, "Config file path")
	flag.StringVar(&stateRecordsConfigFilePath, "records", ddrmRecordsConfigFilePath, "Records config file path")
	flag.BoolVar(&stateAllowImpreciseMatch, "imprecise", stateAllowImpreciseMatch, "Allow imprecise string matches")
	flag.BoolVar(&stateExpand, "expand", stateExpand, "Expand tabs and quoted characters in results")
	flag.IntVar(&stateTabsToSpaces, "tabs", stateTabsToSpaces, "Number of spaces to expand tabs to")
	flag.BoolVar(&stateUseRedis, "cache", stateUseRedis, "Use Redis for persistent rolling update cache")
	flag.BoolVar(&stateUI, "ui", stateUI, "Draw the UI")
	flag.BoolVar(&stateLogRecordProcessing, "logrecords", stateLogRecordProcessing, "Log record processing results")
	flag.DurationVar(&stateUITickRate, "uirate", stateUITickRate, "Seconds between UI updates")
	flag.BoolVar(&statePrintVersion, "version", statePrintVersion, "Print version and exit")
	flag.Parse()
}

func runLoop() {
	// If we get here there's either no TUI, or the TUI decided we're quitting
	// If there's no TUI then we don't want to quit, we just want to run without it
	// select {} is a low CPU usage non-blocking operation that yields nicely
	if !stateUI {
		select {}
	}
}

func printStartupBanner() {
	if statePrintVersion {
		fmt.Printf(ddrmStartupBanner, BuildVersion, BuildDate, GitRev, BuildUser)
		os.Exit(ddrmExitOK)
	}

	dbgf(ddrmStartupBanner, BuildVersion, BuildDate, GitRev, BuildUser)
	dbg(ddrmDebugMode)
}

// utility methods to more easily write debug log texts
func dbgf(format string, v ...interface{}) {
	if stateDebug {
		ddrmLog.Debug().Msgf(format, v...)
	}
}

func dbg(msg string) {
	if stateDebug {
		ddrmLog.Debug().Msg(msg)
	}
}

func dbgp(v ...interface{}) {
	if stateDebug {
		ddrmLog.Print(v...)
	}
}
