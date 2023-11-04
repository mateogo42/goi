# GOI
This is an implementation of an encoder/decoder for the `qoi` format. For more information about the format spec see [https://qoiformat.org/qoi-specification.pdf](https://qoiformat.org/qoi-specification.pdf)

## Build
```bash
go build
```

## Usage
Transform a `png` file into a `qoi` file
```bash
goi encode <png_path>

```
Transform a `qoi` file into a `png` file
```bash
goi decode <qoi_image_path>

```
Show benchmarks of the `qoi` encoder/decoder vs. Go's `png` library for images inside a specified directory.
```bash
goi benchmark <directory>

```
