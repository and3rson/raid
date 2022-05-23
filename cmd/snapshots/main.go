package main

import (
	"io/ioutil"

	"github.com/and3rson/raid/raid"
	log "github.com/sirupsen/logrus"
)

func main() {
	updaterState := &raid.UpdaterState{}
	raid.NewUpdater("", nil, 0, updaterState)
	delorean := raid.NewDelorean("history", nil)
	mapGenerator := raid.NewMapGenerator(updaterState, nil)

	records, err := delorean.ListRecords()
	if err != nil {
		log.Fatal(err)
	}

	for index, record := range records {
		state := updaterState.FindState(record.StateID)
		state.Alert = record.Alert

		log.Infof("main: render image %d/%d", index+1, len(records))

		if err := mapGenerator.GenerateMap(updaterState, record.Date.Format("02.01.2006"), false); err != nil {
			log.Fatal(err)
		}

		filename := record.Date.Format("snapshots/2006-01-02T15_04_05.png")

		if err := ioutil.WriteFile(filename, mapGenerator.MapData.Bytes, 0o644); err != nil {
			log.Fatal(err)
		}
	}
}
