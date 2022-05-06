package raid

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
	log "github.com/sirupsen/logrus"
)

type Delorean struct {
	db            *sql.DB
	addRecordStmt *sql.Stmt
	updates       *Topic[Update]
}

type Record struct {
	ID      int       `json:"id"`
	Date    time.Time `json:"date"`
	StateID int       `json:"state_id"`
	Alert   bool      `json:"alert"`
}

func NewDelorean(dbname string, updates *Topic[Update]) *Delorean {
	db, err := sql.Open("sqlite3", fmt.Sprintf("./data/%s.sqlite", dbname))
	if err != nil {
		log.Fatalf("delorean: open DB: %s", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			date timestamp NOT NULL,
			state_id integer NOT NULL,
			alert bool NOT NULL
		);
	`); err != nil {
		log.Fatalf("delorean: execute schema mutation: %s", err)
	}

	addRecordStmt, err := db.Prepare(`
		INSERT INTO events (date, state_id, alert)
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

func (d *Delorean) ListRecords() ([]Record, error) {
	rows, err := d.db.Query("SELECT * FROM events ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("delorean: list records: %v", err)
	}
	defer rows.Close()

	result := []Record{}

	for rows.Next() {
		record := Record{}
		if err := rows.Scan(&record.ID, &record.Date, &record.StateID, &record.Alert); err != nil {
			return nil, fmt.Errorf("delorean: scan row: %v", err)
		}

		result = append(result, record)
	}

	return result, nil
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
