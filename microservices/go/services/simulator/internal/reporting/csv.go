package reporting

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OutputReporter defines the contract for emitting simulation telemetry
type ActorReporter interface {
	AddActorAction(householdID, actorID, actionID string, isShared bool, simTime time.Time) error
}

type PowerReporter interface {
	AddPowerUsage(householdID string, simTime time.Time, totalWatts, indoorTempC, tankTempC float64, activeDevices []string) error
}

type MeterReporter interface {
	AddActorMeters(householdID, actorID string, simTime time.Time, energy, hunger, hygiene, leisure float64) error
}

// CSVReporter implements OutputReporter using buffered encoding/csv writers
type CSVReporter struct {
	actorFile *os.File
	actorCsv  *csv.Writer
	powerFile *os.File
	powerCsv  *csv.Writer
	meterFile *os.File
	meterCsv  *csv.Writer
}

func NewCSVReporter(outDir string) (*CSVReporter, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	aFile, err := os.Create(filepath.Join(outDir, "_actor_timeline.csv"))
	if err != nil {
		return nil, err
	}
	aCsv := csv.NewWriter(aFile)
	aCsv.Write([]string{"HouseholdID", "Timestamp", "ActorID", "ActionID", "IsShared"})

	pFile, err := os.Create(filepath.Join(outDir, "_power_usage.csv"))
	if err != nil {
		return nil, err
	}
	pCsv := csv.NewWriter(pFile)
	pCsv.Write([]string{"HouseholdID", "Timestamp", "TotalWatts", "IndoorTempC", "TankTempC", "ActiveDevices"})

	mFile, err := os.Create(filepath.Join(outDir, "_actor_meters.csv"))
	if err != nil {
		return nil, err
	}
	mCsv := csv.NewWriter(mFile)
	mCsv.Write([]string{"HouseholdID", "Timestamp", "ActorID", "Energy", "Hunger", "Hygiene", "Leisure"})

	return &CSVReporter{
		actorFile: aFile,
		actorCsv:  aCsv,
		powerFile: pFile,
		powerCsv:  pCsv,
		meterFile: mFile,
		meterCsv:  mCsv,
	}, nil
}

func (r *CSVReporter) AddActorAction(householdID, actorID, actionID string, isShared bool, simTime time.Time) error {
	sharedStr := "false"
	if isShared {
		sharedStr = "true"
	}
	return r.actorCsv.Write([]string{
		householdID,
		simTime.Format(time.RFC3339),
		actorID,
		actionID,
		sharedStr,
	})
}

func (r *CSVReporter) AddPowerUsage(householdID string, simTime time.Time, totalWatts, indoorTempC, tankTempC float64, activeDevices []string) error {
	return r.powerCsv.Write([]string{
		householdID,
		simTime.Format(time.RFC3339),
		fmt.Sprintf("%.2f", totalWatts),
		fmt.Sprintf("%.2f", indoorTempC),
		fmt.Sprintf("%.2f", tankTempC),
		strings.Join(activeDevices, "|"),
	})
}

func (r *CSVReporter) AddActorMeters(householdID, actorID string, simTime time.Time, energy, hunger, hygiene, leisure float64) error {
	return r.meterCsv.Write([]string{
		householdID,
		simTime.Format(time.RFC3339),
		actorID,
		fmt.Sprintf("%.2f", energy),
		fmt.Sprintf("%.2f", hunger),
		fmt.Sprintf("%.2f", hygiene),
		fmt.Sprintf("%.2f", leisure),
	})
}

func (r *CSVReporter) Close() {
	r.actorCsv.Flush()
	r.actorFile.Close()
	r.powerCsv.Flush()
	r.powerFile.Close()
	r.meterCsv.Flush()
	r.meterFile.Close()
}
