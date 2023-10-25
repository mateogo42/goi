package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var benchmarkCmd = &cobra.Command{
	Use:  "benchmark [ITER] [DIR]",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[1]
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if filepath.Ext(path) == ".png" {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				now := time.Now()
				img, err := png.Decode(file)
				if err != nil {
					return err
				}
				elapsedDecodePng := time.Since(now)
				t := table.NewWriter()
				rowConfigAutoMerge := table.RowConfig{AutoMerge: true}
				t.AppendRow(table.Row{path, path, path, path, path}, rowConfigAutoMerge)
				t.AppendRow(table.Row{"Algorithm", "Decode Time", "Encode Time", "Size (KB)", "Rate"})
				width := img.Bounds().Dx()
				height := img.Bounds().Dy()
				channels := getChannels(img, width, height)

				now = time.Now()
				out, err := os.Create(fmt.Sprintf("%s.qoi", getFileName(path)))
				if err != nil {
					return err
				}
				// ENCODE
				encoded := bufio.NewWriter(out)
				encode(img, encoded)
				encoded.Flush()
				statQoi, _ := out.Stat()
				sizeQoi := statQoi.Size()
				elapsedEncodeQoi := time.Since(now)
				out.Close()
				rateQoi := float64(sizeQoi) / float64(width*height*channels)

				buf := bytes.NewBuffer([]byte{})
				now = time.Now()
				err = png.Encode(buf, img)
				if err != nil {
					return err
				}
				elapsedEncodePng := time.Since(now)
				sizePng := buf.Len()
				ratePng := float64(sizePng) / float64(width*height*channels)
				file.Close()

				// DECODE
				qoiFile, err := os.Open(fmt.Sprintf("%s.qoi", getFileName(path)))
				if err != nil {
					return err
				}
				reader := bufio.NewReader(qoiFile)
				now = time.Now()
				_, err = decode(reader)
				if err != nil {
					return err
				}
				elapsedDecodeQoi := time.Since(now)
				qoiFile.Close()
				t.AppendRow(table.Row{"QOI", elapsedDecodeQoi, elapsedEncodeQoi, float64(sizeQoi) / 1024, fmt.Sprintf("%.1f %%", rateQoi*100)})
				t.AppendRow(table.Row{"PNG", elapsedDecodePng, elapsedEncodePng, float64(sizePng) / 1024, fmt.Sprintf("%.1f %%", ratePng*100)})
				t.SetOutputMirror(os.Stdout)
				t.SetStyle(table.StyleLight)
				t.Style().Options.SeparateRows = true
				t.Render()
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)
}

func getChannels(img image.Image, width, height int) int {
	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			px := color.NRGBAModel.Convert(img.At(i, j)).(color.NRGBA)
			if px.A != 255 {
				return 4
			}
		}
	}

	return 3
}
