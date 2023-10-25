package cmd

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/spf13/cobra"
)

const MASK = 0b11000000

var decodeCmd = &cobra.Command{
	Use: "decode [QOI_IMG]",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		img, err := decode(reader)
		if err != nil {
			return err
		}
		out, err := os.Create(fmt.Sprintf("%s_decoded.png", getFileName(path)))
		if err != nil {
			return err
		}
		defer out.Close()

		return png.Encode(out, img)
	},
}

type QoiImage struct {
	width  int
	height int
	pixels []color.NRGBA
}

func newQoiImage(width, height int) QoiImage {
	pixels := make([]color.NRGBA, width*height)
	return QoiImage{
		width:  width,
		height: height,
		pixels: pixels,
	}
}

func (im QoiImage) ColorModel() color.Model {
	return color.NRGBA64Model
}

func (im QoiImage) Bounds() image.Rectangle {
	min := image.Point{0, 0}
	max := image.Point{im.width, im.height}
	return image.Rectangle{Min: min, Max: max}
}

func (im QoiImage) At(x, y int) color.Color {
	if x >= im.width {
		x = im.width - 1
	}
	if y >= im.height {
		y = im.height - 1
	}

	return im.pixels[y*im.width+x]
}

func readHeader(reader *bufio.Reader) (int, int, error) {
	header := make([]byte, 4)
	err := binary.Read(reader, binary.BigEndian, header)
	if err != nil {
		return 0, 0, err
	}

	widthBuf := make([]byte, 4)
	err = binary.Read(reader, binary.BigEndian, widthBuf)
	if err != nil {
		return 0, 0, err
	}
	width := int(widthBuf[0])<<24 | int(widthBuf[1])<<16 | int(widthBuf[2])<<8 | int(widthBuf[3])
	heightBuf := make([]byte, 4)
	err = binary.Read(reader, binary.BigEndian, heightBuf)
	if err != nil {
		return 0, 0, err
	}
	height := int(heightBuf[0])<<24 | int(heightBuf[1])<<16 | int(heightBuf[2])<<8 | int(heightBuf[3])

	channelBuf := make([]byte, 1)
	err = binary.Read(reader, binary.BigEndian, channelBuf)
	if err != nil {
		return 0, 0, err
	}
	colorSpaceBuf := make([]byte, 1)
	err = binary.Read(reader, binary.BigEndian, colorSpaceBuf)
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

func init() {
	rootCmd.AddCommand(decodeCmd)
}

func decode(reader *bufio.Reader) (image.Image, error) {
	width, height, _ := readHeader(reader)
	img := newQoiImage(width, height)
	px := color.NRGBA{0, 0, 0, 255}
	index := [64]color.NRGBA{}
	var run uint8 = 0
	for i := 0; i < height*width-1; i++ {
		if run > 0 {
			run--
		} else {
			b := make([]byte, 1)
			err := binary.Read(reader, binary.BigEndian, b)
			if err != nil {
				return nil, err
			}
			switch {
			case b[0] == QOI_OP_RGB:
				rgb := make([]byte, 3)
				err := binary.Read(reader, binary.BigEndian, rgb)
				if err != nil {
					return nil, err
				}
				px.R = rgb[0]
				px.G = rgb[1]
				px.B = rgb[2]
			case b[0] == QOI_OP_RGBA:
				rgba := make([]byte, 4)
				err := binary.Read(reader, binary.BigEndian, rgba)
				if err != nil {
					return nil, err
				}
				px.R = rgba[0]
				px.G = rgba[1]
				px.B = rgba[2]
				px.A = rgba[3]
			case b[0]&MASK == QOI_OP_INDEX:
				hashPx := b[0] & 0b00111111
				px = index[hashPx]
			case b[0]&MASK == QOI_OP_DIFF:
				px.R += ((b[0] >> 4) & 0b00000011) - 2
				px.G += ((b[0] >> 2) & 0b00000011) - 2
				px.B += (b[0] & 0b00000011) - 2
			case b[0]&MASK == QOI_OP_LUMA:
				drdb := make([]byte, 1)
				err := binary.Read(reader, binary.BigEndian, drdb)
				if err != nil {
					return nil, err
				}
				dg := (b[0] & 0b00111111) - 32
				px.R += dg - 8 + ((drdb[0] >> 4) & 0b00001111)
				px.G += dg
				px.B += dg - 8 + (drdb[0] & 0b00001111)
			case b[0]&MASK == QOI_OP_RUN:
				run = b[0] & 0b00111111
			}
		}
		index[hashPixel(px)] = px
		img.pixels[i] = px
	}

	return img, nil
}
