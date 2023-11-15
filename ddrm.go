//go:build client
// +build client

package main

// Build time information
var (
	BuildVersion string
	BuildDate    string
	GitRev       string
	BuildUser    string
)

func main() {
	// Initial application setup
	parseCliFlags()
	setupLogger()
	printStartupBanner()

	// Read configuration of app state and records to process
	readAppConfig()
	readRecordsConfig()

	// Re-initialise the Redis cache if needed
	reinitRedis()

	// Run some tests and exit if requested
	sendTestEmail()
	testDnsClient()

	// Setup the TUI if needed, along with the periodic tasks
	setupTui()
	setupPeriodicTasks()

	// Run the record processor once outside the periodic scheduler just to prime the state
	processRecords()

	// Run the TUI, which has its own run loop
	runTui()

	// Run the main non-TUI loop if the TUI isn't needed
	runLoop()
}
