package cmd

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	QOI_OP_INDEX = 0x00
	QOI_OP_DIFF  = 0x40
	QOI_OP_LUMA  = 0x80
	QOI_OP_RUN   = 0xC0
	QOI_OP_RGB   = 0xFE
	QOI_OP_RGBA  = 0xFF
)

var encodeCmd = &cobra.Command{
	Use:                   "encode [IMG]",
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		img, err := png.Decode(file)
		if err != nil {
			return err
		}
		out, err := os.Create(fmt.Sprintf("%s.qoi", getFileName(path)))
		encoded := bufio.NewWriter(out)
		if err != nil {
			return err
		}
		defer out.Close()
		encode(img, encoded)

		return encoded.Flush()
	},
}

func init() {
	rootCmd.AddCommand(encodeCmd)
}

func getFileName(fileName string) string {
	ext := filepath.Ext(fileName)
	return fileName[:len(fileName)-len(ext)]
}

func writeHeader(writer *bufio.Writer, width, height int) {
	err := binary.Write(writer, binary.BigEndian, []byte{'q', 'o', 'i', 'f'})
	if err != nil {
		fmt.Println(err)
	}
	err = binary.Write(writer, binary.BigEndian, uint32(width))
	if err != nil {
		fmt.Println(err)
	}
	err = binary.Write(writer, binary.BigEndian, uint32(height))
	if err != nil {
		fmt.Println(err)
	}
	err = binary.Write(writer, binary.BigEndian, []byte{4, 0})
	if err != nil {
		fmt.Println(err)
	}
}

func hashPixel(px color.NRGBA) uint8 {
	return (px.R*3 + px.G*5 + px.B*7 + px.A*11) % 64
}

func encode(img image.Image, encoded *bufio.Writer) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	writeHeader(encoded, width, height)

	var run uint8 = 0
	index := [64]color.NRGBA{}
	prevPx := color.NRGBA{0, 0, 0, 255}

	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			px := color.NRGBAModel.Convert(img.At(i, j)).(color.NRGBA)
			if prevPx == px {
				run++
				if run == 62 || (i == width-1 && j == height-1) {
					err := encoded.WriteByte(QOI_OP_RUN | (run - 1))
					if err != nil {
						fmt.Println(err)
					}
					run = 0
				}
			} else {
				if run > 0 {
					err := encoded.WriteByte(QOI_OP_RUN | (run - 1))
					if err != nil {
						fmt.Println(err)
					}
					run = 0
				}
				hashPx := hashPixel(px)
				if index[hashPx] == px {
					err := encoded.WriteByte(QOI_OP_INDEX | hashPx)
					if err != nil {
						fmt.Println(err)
					}
				} else {
					index[hashPx] = px
					if prevPx.A == px.A {
						dr := int8(int16(px.R) - int16(prevPx.R))
						dg := int8(int16(px.G) - int16(prevPx.G))
						db := int8(int16(px.B) - int16(prevPx.B))

						dr_dg := dr - dg
						db_dg := db - dg

						if (-3 < dr && dr < 2) && (-3 < dg && dg < 2) && (-3 < db && db < 2) {
							err := binary.Write(encoded, binary.BigEndian, uint8(QOI_OP_DIFF|(dr+2)<<4|(dg+2)<<2|(db+2)))
							if err != nil {
								fmt.Println(err)
							}
						} else if (-9 < dr_dg && dr_dg < 8) && (-33 < dg && dg < 32) && (-9 < db_dg && db_dg < 8) {
							err := binary.Write(encoded, binary.BigEndian, []byte{uint8(QOI_OP_LUMA | uint8(dg+32)), uint8((dr_dg+8)<<4 | (db_dg + 8))})
							if err != nil {
								fmt.Println(err)
							}
						} else {
							err := binary.Write(encoded, binary.BigEndian, []byte{QOI_OP_RGB, px.R, px.G, px.B})
							if err != nil {
								fmt.Println(err)
							}
						}
					} else {
						err := binary.Write(encoded, binary.BigEndian, []byte{QOI_OP_RGBA, px.R, px.G, px.B, px.A})
						if err != nil {
							fmt.Println(err)
						}
					}
				}
			}
			prevPx = px
		}
	}

	err := binary.Write(encoded, binary.BigEndian, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		fmt.Println(err)
	}
}
