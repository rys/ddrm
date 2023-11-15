//go:build client
// +build client

package main

import (
	"fmt"
	"slices"
	"strings"
)

// Type to store currently fetched record state
type DdrmRecordState struct {
	FQDN          string
	Type          DdrmRecordType
	CurrentValues []string
	PriorValues   []string
	Changed       bool
	SentEmail     bool
	Errored       bool
	Processing    bool
}

// current in-memory record states
var ddrmRecordStates map[string]DdrmRecordState

func checkRecordDataForChanges(fqdn string, recordType DdrmRecordType, answer []string) (changed bool, fetched []string, cached []string, compare int) {
	changed = false

	cache := getCachedValues(fqdn, recordType)

	expectedConfigs := make(map[string]DdrmRecordConfig)

	for _, recordConfig := range ddrmRecordConfig {
		expectedConfigs[recordConfig.FQDN+":"+string(recordConfig.Type)] = recordConfig
	}

	if len(cache) == 0 {
		// fall back to the startup config
		cache = expectedConfigs[fqdn+":"+string(recordType)].ExpectedValues
	}

	// sort both and compare
	slices.Sort(cache)
	slices.Sort(answer)

	compare = slices.Compare(cache, answer)
	changed = compare != 0
	fetched = answer
	cached = cache

	_ = setCachedValues(fqdn, recordType, answer)

	return
}

func stringProcessor(s string) string {
	if stateExpand {
		s = strings.ReplaceAll(s, "\t", strings.Repeat(" ", stateTabsToSpaces))
		return s
	}
	if stateAllowImpreciseMatch {
		s = strings.TrimSuffix(s, ".")
	}

	return s
}

func processRecords() {
	for _, record := range ddrmRecordConfig {
		// indicate processing state for everything
		state := ddrmRecordStates[record.FQDN+":"+string(record.Type)]
		state.Processing = true
		state.SentEmail = false
		state.Errored = false
		state.Changed = false
		ddrmRecordStates[record.FQDN+":"+string(record.Type)] = state
	}

	for _, record := range ddrmRecordConfig {
		data := getRecordData(record.FQDN, record.Type)

		if len(data) == 0 {
			dbgf(ddrmErrorUnableToFetchRecord, record.FQDN, string(record.Type))

			// set the error state but don't remove the the current + expected so they can be shown in the UI
			state := ddrmRecordStates[record.FQDN+":"+string(record.Type)]
			state.Errored = true
			state.Processing = false
			ddrmRecordStates[record.FQDN+":"+string(record.Type)] = state
		} else {
			changed, fetched, cached, compare := checkRecordDataForChanges(record.FQDN, record.Type, data)

			// update the running state
			state := ddrmRecordStates[record.FQDN+":"+string(record.Type)]
			state.Changed = changed
			state.CurrentValues = fetched
			state.PriorValues = cached
			state.Errored = false
			state.FQDN = record.FQDN
			state.Type = record.Type
			state.Processing = false

			if stateLogRecordProcessing {
				dbg("changed = " + fmt.Sprint(changed))
				dbg("compare = " + fmt.Sprint(compare))
				dbg("fetched = " + fmt.Sprint(fetched))
				dbg("cached  = " + fmt.Sprint(cached))
				dbg("data    = " + fmt.Sprint(data))
			}
			if changed {
				state.SentEmail = sendEmail(record.FQDN, record.Type, fetched, cached)
			}

			ddrmRecordStates[record.FQDN+":"+string(record.Type)] = state
		}
		dbg("")
	}
}
