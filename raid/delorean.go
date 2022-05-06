package raid

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
	log "github.com/sirupsen/logrus"
)

type Delorean struct {
	db            *sql.DB
	addRecordStmt *sql.Stmt
	updates       *Topic[Update]
}

func NewDelorean(dbname string, updates *Topic[Update]) *Delorean {
	db, err := sql.Open("sqlite3", fmt.Sprintf("./data/%s.sqlite", dbname))
	if err != nil {
		log.Fatalf("delorean: open DB: %s", err)
	}

	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS events (
			id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			date timestamp NOT NULL,
			state integer NOT NULL,
			alert bool NOT NULL
		);
	`)
	if err != nil {
		log.Fatalf("delorean: prepare schema mutation: %s", err)
	}

	if _, err := stmt.Exec(); err != nil {
		log.Fatalf("delorean: execute schema mutation: %s", err)
	}

	addRecordStmt, err := db.Prepare(`
		INSERT INTO events (date, state, alert)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("delorean: prepare add record: %s", err)
	}

	return &Delorean{db, addRecordStmt, updates}
}

func (d *Delorean) addRecord(state State) error {
	if _, err := d.addRecordStmt.Exec(state.Changed, state.ID, state.Alert); err != nil {
		return fmt.Errorf("delorean: execute add record: %v", err)
	}

	return nil
}

func (d *Delorean) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer log.Debug("delorean: exit")

	defer wg.Done()
	wg.Add(1)

	events := d.updates.Subscribe("delorean", FilterAll[Update])
	defer d.updates.Unsubscribe(events)

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}

			if err := d.addRecord(event.State); err != nil {
				errch <- fmt.Errorf("delorean: add record: %s", err)

				return
			}
		case <-ctx.Done():
			return
		}
	}
}
