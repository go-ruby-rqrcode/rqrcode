// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import (
	"strconv"
	"strings"
)

// SVGOptions configures AsSVG, mirroring rqrcode's as_svg option hash.
type SVGOptions struct {
	Offset         int    // padding around the QR in pixels (default 0)
	OffsetX        int    // X padding; falls back to Offset when OffsetXSet is false
	OffsetXSet     bool   // whether OffsetX was explicitly provided
	OffsetY        int    // Y padding; falls back to Offset when OffsetYSet is false
	OffsetYSet     bool   // whether OffsetY was explicitly provided
	Fill           string // background color hex (no leading #); empty = none
	FillNamed      bool   // treat Fill as a named color (no "#" prefix)
	Color          string // foreground color hex (default "000")
	ColorNamed     bool   // treat Color as a named color (no "#" prefix)
	ShapeRendering string // default "crispEdges"
	ModuleSize     int    // pixel size of each module (default 11)
	Standalone     *bool  // full SVG file when true/nil; embeddable svg when false
	Viewbox        bool   // use viewBox instead of width/height
	UsePath        bool   // render with <path> instead of <rect>
	SVGAttributes  [][2]string
}

// AsSVG renders the QR code as an SVG document, matching RQRCode's as_svg. The
// zero-value options reproduce the gem defaults.
func (q *QRCode) AsSVG(opts SVGOptions) string {
	offset := opts.Offset
	offsetX := offset
	if opts.OffsetXSet {
		offsetX = opts.OffsetX
	}
	offsetY := offset
	if opts.OffsetYSet {
		offsetY = opts.OffsetY
	}
	color := opts.Color
	if color == "" {
		color = "000"
	}
	shapeRendering := opts.ShapeRendering
	if shapeRendering == "" {
		shapeRendering = "crispEdges"
	}
	moduleSize := opts.ModuleSize
	if moduleSize == 0 {
		moduleSize = 11
	}
	standalone := true
	if opts.Standalone != nil {
		standalone = *opts.Standalone
	}

	width := q.ModuleCount*moduleSize + 2*offsetX
	height := q.ModuleCount*moduleSize + 2*offsetY
	dimension := width
	if height > dimension {
		dimension = height
	}

	var dimensionsAttr string
	if opts.Viewbox {
		dimensionsAttr = `viewBox="0 0 ` + itoa(width) + " " + itoa(height) + `"`
	} else {
		dimensionsAttr = `width="` + itoa(width) + `" height="` + itoa(height) + `"`
	}

	attrs := append([]string{}, defaultSVGAttributes...)
	attrs = append(attrs, dimensionsAttr, `shape-rendering="`+shapeRendering+`"`)
	for _, kv := range opts.SVGAttributes {
		attrs = append(attrs, kv[0]+`="`+kv[1]+`"`)
	}
	svgTagAttributes := strings.Join(attrs, " ")

	xmlTag := `<?xml version="1.0" standalone="yes"?>`
	openTag := `<svg ` + svgTagAttributes + `>`
	closeTag := `</svg>`

	// Prefix hexadecimal colors unless using a named color.
	if !opts.ColorNamed {
		color = "#" + color
	}

	var result []string
	if opts.UsePath {
		result = q.svgPath(moduleSize, color, offsetX, offsetY)
	} else {
		result = q.svgRect(moduleSize, color, offsetX, offsetY)
	}

	if opts.Fill != "" || opts.FillNamed {
		fill := opts.Fill
		if !opts.FillNamed {
			fill = "#" + fill
		}
		rect := `<rect width="` + itoa(dimension) + `" height="` + itoa(dimension) +
			`" x="0" y="0" fill="` + fill + `"/>`
		result = append([]string{rect}, result...)
	}

	if standalone {
		result = append([]string{xmlTag, openTag}, result...)
		result = append(result, closeTag)
	}
	return strings.Join(result, "")
}

// defaultSVGAttributes is RQRCode::Export::SVG::DEFAULT_SVG_ATTRIBUTES.
var defaultSVGAttributes = []string{
	`version="1.1"`,
	`xmlns="http://www.w3.org/2000/svg"`,
	`xmlns:xlink="http://www.w3.org/1999/xlink"`,
	`xmlns:ev="http://www.w3.org/2001/xml-events"`,
}

// svgRect renders modules as <rect> elements (SVG::Rect#build).
func (q *QRCode) svgRect(moduleSize int, color string, offsetX, offsetY int) []string {
	var result []string
	// Ruby iterates modules.each_index (c) then each_index (r) and uses
	// checked?(c, r) — the outer index is the row, inner is the column, but the
	// pixel x uses r and y uses c.
	for c := 0; c < q.ModuleCount; c++ {
		for r := 0; r < q.ModuleCount; r++ {
			if !q.Modules[c][r] {
				continue
			}
			x := r*moduleSize + offsetX
			y := c*moduleSize + offsetY
			result = append(result, `<rect width="`+itoa(moduleSize)+`" height="`+itoa(moduleSize)+
				`" x="`+itoa(x)+`" y="`+itoa(y)+`" fill="`+color+`"/>`)
		}
	}
	return result
}

// Direction constants for the path renderer (SVG::Path).
const (
	dirUp    = 0
	dirDown  = 1
	dirLeft  = 2
	dirRight = 3
)

// dirDeltas is SVG::Path::DIR_DELTAS.
var dirDeltas = [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

// dirPathCommands is SVG::Path::DIR_PATH_COMMANDS.
var dirPathCommands = [4]string{"v-", "v", "h-", "h"}

// edge is one boundary segment [x, y, direction].
type edge struct{ x, y, dir int }

// svgPath renders modules as a single <path> (SVG::Path#build).
func (q *QRCode) svgPath(moduleSize int, color string, offsetX, offsetY int) []string {
	modules := q.Modules
	moduleCount := len(modules)
	matrixSize := moduleCount + 1

	edgeMatrix := make([][][]*edge, matrixSize)
	for i := range edgeMatrix {
		edgeMatrix[i] = make([][]*edge, matrixSize)
	}
	edgeCount := 0

	// Horizontal edges (between vertically adjacent cells).
	for rowIndex := 0; rowIndex <= moduleCount; rowIndex++ {
		for colIndex := 0; colIndex < moduleCount; colIndex++ {
			above := rowIndex > 0 && modules[rowIndex-1][colIndex]
			below := rowIndex < moduleCount && modules[rowIndex][colIndex]
			if above && !below {
				x, y := colIndex+1, rowIndex
				edgeMatrix[y][x] = append(edgeMatrix[y][x], &edge{x, y, dirLeft})
				edgeCount++
			} else if !above && below {
				x, y := colIndex, rowIndex
				edgeMatrix[y][x] = append(edgeMatrix[y][x], &edge{x, y, dirRight})
				edgeCount++
			}
		}
	}

	// Vertical edges (between horizontally adjacent cells).
	for rowIndex := 0; rowIndex < moduleCount; rowIndex++ {
		for colIndex := 0; colIndex <= moduleCount; colIndex++ {
			left := colIndex > 0 && modules[rowIndex][colIndex-1]
			right := colIndex < moduleCount && modules[rowIndex][colIndex]
			if left && !right {
				x, y := colIndex, rowIndex
				edgeMatrix[y][x] = append(edgeMatrix[y][x], &edge{x, y, dirDown})
				edgeCount++
			} else if !left && right {
				x, y := colIndex, rowIndex+1
				edgeMatrix[y][x] = append(edgeMatrix[y][x], &edge{x, y, dirUp})
				edgeCount++
			}
		}
	}

	var pathParts []string
	searchY, searchX := 0, 0

	for edgeCount > 0 {
		var startEdge *edge
		foundY, foundX := searchY, searchX
		for y := searchY; y < matrixSize; y++ {
			startCol := 0
			if y == searchY {
				startCol = searchX
			}
			for x := startCol; x < matrixSize; x++ {
				cell := edgeMatrix[y][x]
				if len(cell) == 0 {
					continue
				}
				startEdge = cell[0]
				foundY, foundX = y, x
				break
			}
			if startEdge != nil {
				break
			}
		}
		searchY, searchX = foundY, foundX

		var sb strings.Builder
		sb.WriteString("M")
		sb.WriteString(itoa(startEdge.x))
		sb.WriteString(" ")
		sb.WriteString(itoa(startEdge.y))

		currentEdge := startEdge
		currentDir := -1
		currentCount := 0

		for currentEdge != nil {
			ex, ey, edir := currentEdge.x, currentEdge.y, currentEdge.dir

			// Remove edge from matrix.
			cell := edgeMatrix[ey][ex]
			edgeMatrix[ey][ex] = removeEdge(cell, currentEdge)
			edgeCount--

			if edir == currentDir {
				currentCount++
			} else {
				if currentDir != -1 {
					sb.WriteString(dirPathCommands[currentDir])
					sb.WriteString(itoa(currentCount))
				}
				currentDir = edir
				currentCount = 1
			}

			delta := dirDeltas[edir]
			nextX, nextY := ex+delta[0], ey+delta[1]
			currentEdge = nil
			if nextY >= 0 && nextY < matrixSize && nextX >= 0 && nextX < matrixSize {
				if nc := edgeMatrix[nextY][nextX]; len(nc) > 0 {
					currentEdge = nc[0]
				}
			}
		}

		sb.WriteString("z")
		pathParts = append(pathParts, sb.String())
	}

	path := `<path d="` + strings.Join(pathParts, "") + `" fill="` + color +
		`" transform="translate(` + itoa(offsetX) + `,` + itoa(offsetY) + `) scale(` + itoa(moduleSize) + `)"/>`
	return []string{path}
}

// removeEdge deletes the first occurrence of e from cell (Ruby's Array#delete of
// the identical object), returning nil when the cell becomes empty.
func removeEdge(cell []*edge, e *edge) []*edge {
	out := cell[:0]
	for _, c := range cell {
		if c != e {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// itoa is a short alias for strconv.Itoa used throughout the renderers.
func itoa(i int) string { return strconv.Itoa(i) }
