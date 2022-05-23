package raid

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"sync"
	"text/template"

	"github.com/golang/freetype"
	log "github.com/sirupsen/logrus"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
)

//go:embed assets/ua.svg.tpl
var mapTemplateStr string

//go:embed assets/DejaVuSansMono-Bold.ttf
var dejaVuFontData []byte

const MapWidth = 1000
const MapHeight = 670

type MapData struct {
	ContentType string
	Bytes       []byte
}

type MapGenerator struct {
	updaterState *UpdaterState
	updates      *Topic[Update]
	mapTemplate  *template.Template
	fontContext  *freetype.Context
	MapData      *MapData
}

func NewMapGenerator(updaterState *UpdaterState, updates *Topic[Update]) *MapGenerator {
	mapTemplate, err := template.New("maptemplate").Parse(mapTemplateStr)
	if err != nil {
		log.Fatalf("mapgenerator: parse map template: %s", err)
	}

	f, err := freetype.ParseFont(dejaVuFontData)
	if err != nil {
		log.Fatalf("mapgenerator: parse TTF font: %s", err)
	}

	fontCtx := freetype.NewContext()
	fontCtx.SetDPI(72)
	fontCtx.SetFont(f)
	fontCtx.SetFontSize(48)
	fontCtx.SetSrc(image.Black)
	fontCtx.SetHinting(font.HintingFull)

	g := &MapGenerator{updaterState, updates, mapTemplate, fontCtx, &MapData{}}
	if err := g.GenerateMap(updaterState, "", true); err != nil {
		log.Fatalf("mapgenerator: generate initial map: %s", err)
	}

	return g
}

func (g *MapGenerator) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer log.Debug("mapgenerator: exit")

	defer wg.Done()
	wg.Add(1)

	events := g.updates.Subscribe("mapgenerator", func(u Update) bool {
		return u.IsFresh
	})
	defer g.updates.Unsubscribe(events)

	for {
		select {
		case _, ok := <-events:
			if !ok {
				return
			}

			if err := g.GenerateMap(g.updaterState, "", true); err != nil {
				errch <- fmt.Errorf("mapgenerator: regenerate map: %w", err)

				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (g *MapGenerator) GenerateMap(updaterState *UpdaterState, title string, transparent bool) error {
	stateAlerts := map[int]bool{}
	for _, state := range updaterState.States {
		stateAlerts[state.ID] = state.Alert
	}

	mapStr := bytes.NewBuffer(nil)
	if err := g.mapTemplate.Execute(mapStr, map[string]interface{}{"alerts": stateAlerts}); err != nil {
		return fmt.Errorf("mapgenerator: execute map template: %w", err)
	}

	svg, _ := oksvg.ReadIconStream(mapStr)
	svg.SetTarget(0, 0, MapWidth, MapHeight)
	rect := image.Rect(0, 0, MapWidth, MapHeight)
	rgba := image.NewRGBA(rect)

	if !transparent {
		draw.Draw(rgba, rect, image.White, image.Point{}, draw.Src)
	}

	svg.Draw(rasterx.NewDasher(MapWidth, MapHeight, rasterx.NewScannerGV(MapWidth, MapHeight, rgba, rgba.Bounds())), 1)

	if len(title) > 0 {
		g.fontContext.SetClip(rgba.Bounds())
		g.fontContext.SetDst(rgba)

		if _, err := g.fontContext.DrawString(title, freetype.Pt(50, 500)); err != nil {
			return fmt.Errorf("mapgenerator: draw text: %w", err)
		}
	}

	out := bytes.NewBuffer(nil)
	if err := png.Encode(out, rgba); err != nil {
		return fmt.Errorf("mapgenerator: encode png map: %w", err)
	}

	g.MapData.ContentType = "image/png"
	g.MapData.Bytes = out.Bytes()

	log.Infof("mapgenerator: generate map complete, size = %d B", len(g.MapData.Bytes))

	return nil
}
