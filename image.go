package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"image"
	"image/png"
	"sync"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

//go:embed assets/ua.svg.tpl
var mapTemplateStr string

const MapWidth = 1000
const MapHeight = 670

type MapData struct {
	ContentType string
	Bytes       []byte
}

type MapGenerator struct {
	updaterState *UpdaterState
	updates      *Topic[*State]
	MapData      *MapData
}

func NewMapGenerator(updaterState *UpdaterState, updates *Topic[*State]) *MapGenerator {
	g := &MapGenerator{updaterState, updates, &MapData{}}
	if err := g.generateMap(updaterState); err != nil {
		log.Fatalf("image: generate initial map: %s", err)
	}

	return g
}

func (g *MapGenerator) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer wg.Done()
	wg.Add(1)

	events := g.updates.Subscribe(FilterAll[State])
	defer g.updates.Unsubscribe(events)

	for {
		select {
		case <-events:
			if err := g.generateMap(g.updaterState); err != nil {
				errch <- fmt.Errorf("image: regenerate map: %s", err)

				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (g *MapGenerator) generateMap(updaterState *UpdaterState) error {
	mapTemplate, err := template.New("maptemplate").Parse(mapTemplateStr)
	if err != nil {
		return fmt.Errorf("image: parse map template: %v", err)
	}

	stateAlerts := map[int]bool{}
	for _, state := range updaterState.States {
		stateAlerts[state.ID] = state.Alert
	}

	mapStr := bytes.NewBuffer(nil)
	if err := mapTemplate.Execute(mapStr, stateAlerts); err != nil {
		return fmt.Errorf("image: execute map template: %v", err)
	}

	svg, _ := oksvg.ReadIconStream(mapStr)
	svg.SetTarget(0, 0, MapWidth, MapHeight)
	rgba := image.NewRGBA(image.Rect(0, 0, MapWidth, MapHeight))
	svg.Draw(rasterx.NewDasher(MapWidth, MapHeight, rasterx.NewScannerGV(MapWidth, MapHeight, rgba, rgba.Bounds())), 1)

	out := bytes.NewBuffer(nil)
	if err := png.Encode(out, rgba); err != nil {
		return fmt.Errorf("image: encode png map: %v", err)
	}

	g.MapData.ContentType = "image/png"
	g.MapData.Bytes = out.Bytes()

	return nil
}
