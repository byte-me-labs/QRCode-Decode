package qrcode

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
	"gocv.io/x/gocv/contrib"
)

type Decoder struct {
	detProto, detModel, srProto, srModel string
}

func NewDecoder(detectProto, detectModel, srProto, srModel string) *Decoder {
	return &Decoder{
		detProto: detectProto,
		detModel: detectModel,
		srProto:  srProto,
		srModel:  srModel,
	}
}

func (d *Decoder) Decode(filePath string) (string, error) {
	img := gocv.IMRead(filePath, gocv.IMReadColor)
	if img.Empty() {
		return "", fmt.Errorf("unable to read image: %s", filePath)
	}
	defer img.Close()
	return d.decode(img)
}

func (d *Decoder) DecodeFromBytes(data []byte) (string, error) {
	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return "", fmt.Errorf("unable to decode image data: %w", err)
	}
	if mat.Empty() {
		return "", fmt.Errorf("decoded image is empty")
	}
	defer mat.Close()
	return d.decode(mat)
}

func (d *Decoder) decode(img gocv.Mat) (string, error) {
	qr := contrib.NewWeChatQRCode(d.detProto, d.detModel, d.srProto, d.srModel)

	if rect, ok := findQRRegion(img); ok {
		for _, ps := range []int{1, 3, 6} {
			pad := (rect.Max.X - rect.Min.X) * ps / 14
			padded := image.Rect(
				max(0, rect.Min.X-pad), max(0, rect.Min.Y-pad),
				min(img.Cols(), rect.Max.X+pad), min(img.Rows(), rect.Max.Y+pad),
			)
			crop := img.Region(padded)
			if text := quickDetect(qr, crop); text != "" {
				return text, nil
			}
		}
	}

	if text := quickDetect(qr, img); text != "" {
		return text, nil
	}
	return "", fmt.Errorf("no QR code detected")
}

func quickDetect(qr *contrib.WeChatQRCode, mat gocv.Mat) string {
	var points []gocv.Mat
	results := qr.DetectAndDecode(mat, &points)
	if len(results) > 0 {
		return results[0]
	}
	return ""
}

// ==================== QR finder pattern detection ====================

type finderPoint struct{ x, y int }

func findQRRegion(img gocv.Mat) (image.Rectangle, bool) {
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	W, H := gray.Cols(), gray.Rows()

	var hits []finderPoint
	step := 2
	for y := step; y < H-step; y += step {
		if c := findFinderCross(gray, W, H, y, true); c.x != 0 {
			hits = append(hits, c)
		}
	}
	for x := step; x < W-step; x += step {
		if c := findFinderCross(gray, W, H, x, false); c.x != 0 {
			hits = append(hits, c)
		}
	}
	if len(hits) < 3 {
		return image.Rectangle{}, false
	}

	const clusterRadius = 30
	type cluster struct{ sx, sy, count int }
	var clusters []cluster
	used := make([]bool, len(hits))
	for i, h := range hits {
		if used[i] {
			continue
		}
		c := cluster{sx: h.x, sy: h.y, count: 1}
		used[i] = true
		for j := i + 1; j < len(hits); j++ {
			if used[j] {
				continue
			}
			dx, dy := hits[j].x-h.x, hits[j].y-h.y
			if dx*dx+dy*dy < clusterRadius*clusterRadius {
				c.sx += hits[j].x
				c.sy += hits[j].y
				c.count++
				used[j] = true
			}
		}
		c.sx /= c.count
		c.sy /= c.count
		clusters = append(clusters, c)
	}
	if len(clusters) < 3 {
		return image.Rectangle{}, false
	}

	type pt struct{ x, y int }
	var bestTL, bestTR, bestBL pt
	bestScore := -1.0
	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			for k := j + 1; k < len(clusters); k++ {
				pts := []pt{
					{clusters[i].sx, clusters[i].sy},
					{clusters[j].sx, clusters[j].sy},
					{clusters[k].sx, clusters[k].sy},
				}
				for a := 0; a < 3; a++ {
					for b := a + 1; b < 3; b++ {
						if (pts[a].x + pts[a].y) > (pts[b].x + pts[b].y) {
							pts[a], pts[b] = pts[b], pts[a]
						}
					}
				}
				a, b, c := pts[0], pts[1], pts[2]
				var tr, bl pt
				if b.x > c.x {
					tr, bl = b, c
				} else {
					tr, bl = c, b
				}

				dxTR, dyTR := tr.x-a.x, tr.y-a.y
				dxBL, dyBL := bl.x-a.x, bl.y-a.y
				if dxTR < 20 || dyBL < 20 || iabs(dyTR)*3 > iabs(dxTR) || iabs(dxBL)*3 > iabs(dyBL) {
					continue
				}
				lng, sht := max(dxTR, dyBL), min(dxTR, dyBL)
				if sht*10 < lng*6 {
					continue
				}

				score := float64(dyTR*dyTR)/float64(dxTR*dxTR) +
					float64(dxBL*dxBL)/float64(dyBL*dyBL) +
					float64(iabs(dxTR-dyBL))/float64(max(dxTR, dyBL))
				if bestScore < 0 || score < bestScore {
					bestScore = score
					bestTL, bestTR, bestBL = a, tr, bl
				}
			}
		}
	}
	if bestScore < 0 {
		return image.Rectangle{}, false
	}

	w, h := bestTR.x-bestTL.x, bestBL.y-bestTL.y
	ms := max(1, min(w, h)/40)

	minX := max(0, bestTL.x-ms*4)
	minY := max(0, bestTL.y-ms*4)
	maxX := min(W, bestTR.x+ms*4)
	maxY := min(H, bestBL.y+ms*4)

	const minW, minH = 700, 500
	if maxX-minX < minW {
		ext := (minW - (maxX - minX)) / 2
		minX, maxX = max(0, minX-ext), min(W, maxX+ext)
	}
	if maxY-minY < minH {
		ext := (minH - (maxY - minY)) / 2
		minY, maxY = max(0, minY-ext), min(H, maxY+ext)
	}
	return image.Rect(minX, minY, maxX, maxY), true
}

func findFinderCross(gray gocv.Mat, W, H, axisIdx int, horizontal bool) finderPoint {
	const maxLen = 2048
	var vals [maxLen]uint8
	count := 0
	if horizontal {
		if axisIdx < 0 || axisIdx >= H {
			return finderPoint{}
		}
		count = min(W, maxLen)
		for x := 0; x < count; x++ {
			vals[x] = gray.GetUCharAt(axisIdx, x)
		}
	} else {
		if axisIdx < 0 || axisIdx >= W {
			return finderPoint{}
		}
		count = min(H, maxLen)
		for y := 0; y < count; y++ {
			vals[y] = gray.GetUCharAt(y, axisIdx)
		}
	}

	threshold := uint8(128)
	runLen, runDark := 0, vals[0] < threshold
	var runs []int
	for i := 0; i < count; i++ {
		if dark := vals[i] < threshold; dark == runDark {
			runLen++
		} else {
			runs = append(runs, runLen)
			runDark, runLen = dark, 1
		}
	}
	runs = append(runs, runLen)
	if len(runs) < 5 {
		return finderPoint{}
	}

	start := 0
	if vals[0] >= threshold {
		start = 1
	}
	for i := start; i+4 < len(runs); i += 2 {
		b1, w1, b2, w2, b3 := runs[i], runs[i+1], runs[i+2], runs[i+3], runs[i+4]
		mod := float64(b1+w1+b2+w2+b3) / 7.0
		if mod < 1 {
			continue
		}
		tol := mod * 0.4
		if fabs(float64(b1)-mod) > tol || fabs(float64(w1)-mod) > tol ||
			fabs(float64(b2)-3*mod) > tol*1.5 || fabs(float64(w2)-mod) > tol ||
			fabs(float64(b3)-mod) > tol {
			continue
		}

		offset := 0
		for j := 0; j < i+2; j++ {
			offset += runs[j]
		}
		offset += runs[i+2] / 2

		if horizontal {
			if offset > 0 && offset < W && axisIdx > 0 && axisIdx < H {
				if checkVertical(gray, W, H, offset, axisIdx, mod) {
					return finderPoint{offset, axisIdx}
				}
			}
		} else {
			if axisIdx > 0 && axisIdx < W && offset > 0 && offset < H {
				if checkHorizontal(gray, W, H, axisIdx, offset, mod) {
					return finderPoint{axisIdx, offset}
				}
			}
		}
	}
	return finderPoint{}
}

func checkVertical(gray gocv.Mat, W, H, cx, cy int, mod float64) bool {
	hs := int(mod * 5)
	y1, y2 := max(0, cy-hs), min(H-1, cy+hs)
	vals := make([]uint8, y2-y1+1)
	for y := y1; y <= y2; y++ {
		vals[y-y1] = gray.GetUCharAt(y, cx)
	}
	return checkRatio(vals)
}

func checkHorizontal(gray gocv.Mat, W, H, cx, cy int, mod float64) bool {
	hs := int(mod * 5)
	x1, x2 := max(0, cx-hs), min(W-1, cx+hs)
	vals := make([]uint8, x2-x1+1)
	for x := x1; x <= x2; x++ {
		vals[x-x1] = gray.GetUCharAt(cy, x)
	}
	return checkRatio(vals)
}

func checkRatio(vals []uint8) bool {
	if len(vals) < 10 {
		return false
	}
	threshold := uint8(128)
	var runs []int
	runLen, runDark := 1, vals[0] < threshold
	for i := 1; i < len(vals); i++ {
		if dark := vals[i] < threshold; dark == runDark {
			runLen++
		} else {
			runs = append(runs, runLen)
			runDark, runLen = dark, 1
		}
	}
	runs = append(runs, runLen)
	if len(runs) < 5 {
		return false
	}
	start := 0
	if vals[0] >= threshold {
		start = 1
	}
	for i := start; i+4 < len(runs); i += 2 {
		b1, w1, b2, w2, b3 := runs[i], runs[i+1], runs[i+2], runs[i+3], runs[i+4]
		mod := float64(b1+w1+b2+w2+b3) / 7.0
		if mod < 1 {
			continue
		}
		tol := mod * 0.4
		if fabs(float64(b1)-mod) > tol || fabs(float64(w1)-mod) > tol ||
			fabs(float64(b2)-3*mod) > tol*1.5 || fabs(float64(w2)-mod) > tol ||
			fabs(float64(b3)-mod) > tol {
			continue
		}
		return true
	}
	return false
}

func iabs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func fabs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
