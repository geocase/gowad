package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
)

type wadinfo_t struct {
	Identification [4]byte
	Numlumps       int32
	Infotableofs   int32
}

type filelump_t struct {
	Filepos int32
	Size    int32
	Name    [8]byte
}

type sprite_lump_t struct {
	Width     uint16
	Height    uint16
	Left_ofst int16
	Top_ofst  int16
}

type wadfile_t struct {
	wadinfo   wadinfo_t
	directory map[string]filelump_t
	raw_file  []byte
}

func LoadWadFromPath(path string) (wadfile_t, error) {
	fmt.Println(path)
	w := wadfile_t{}
	w.wadinfo, _ = LoadWadHeader(path)

	raw_dir := make([]filelump_t, w.wadinfo.Numlumps)
	w.directory = make(map[string]filelump_t)
	// playpal_lump_position := filelump_t{}
	f, _ := os.Open(path)
	f.Seek(int64(w.wadinfo.Infotableofs), 0)
	binary.Read(f, binary.LittleEndian, &raw_dir)
	for i := 0; i < len(raw_dir); i++ {
		clean := strings.Trim(string(raw_dir[i].Name[:]), string(0))
		w.directory[clean] = raw_dir[i]
	}
	f.Seek(0, 0)
	file_stat, _ := f.Stat()
	file_size := file_stat.Size()
	w.raw_file = make([]byte, file_size)
	binary.Read(f, binary.LittleEndian, &w.raw_file)
	f.Close()
	return w, nil
}

func (f filelump_t) InfoString() string {
	cutset := string(0)
	ret := strings.Trim(string(f.Name[:]), cutset)

	return ret
}

func LoadWadHeader(path string) (wadinfo_t, error) {
	f, err := os.Open(path)
	if err != nil {
		return wadinfo_t{[4]byte{0}, 0, 0}, errors.New("Unable to load file: \"" + path + "\"")
	}
	wad_info := wadinfo_t{}

	binary.Read(f, binary.LittleEndian, &wad_info)

	if wad_info.Identification != [4]byte{'P', 'W', 'A', 'D'} && wad_info.Identification != [4]byte{'I', 'W', 'A', 'D'} {
		return wadinfo_t{[4]byte{0}, 0, 0}, errors.New("\"" + path + "\" is not a valid .wad file.")
	}
	f.Close()
	return wad_info, nil
}

func (w wadfile_t) DecodePlaypal(key string) *image.RGBA {
	lump := w.directory[key]
	palette_slice := w.raw_file[lump.Filepos : lump.Filepos+lump.Size]
	ret := image.NewRGBA(image.Rect(0, 0, 256, 1))
	for x := 0; x < len(palette_slice)/3; x++ {
		i := x * 3
		ret.Set(x, 0, color.NRGBA{
			R: palette_slice[i+0],
			G: palette_slice[i+1],
			B: palette_slice[i+2],
			A: 255,
		})
	}
	return ret
}

func (w wadfile_t) DecodeImage(key string) *image.RGBA {
	playpal := w.DecodePlaypal("PLAYPAL")
	lump := w.directory[key]
	sprite := sprite_lump_t{}
	sprite.Width = binary.LittleEndian.Uint16(w.raw_file[lump.Filepos : lump.Filepos+2])
	sprite.Height = binary.LittleEndian.Uint16(w.raw_file[lump.Filepos+2 : lump.Filepos+2+2])
	lump_column_offsets := make([]uint32, sprite.Width)
	for i := 0; i < int(sprite.Width); i++ {
		pos := int(lump.Filepos) + 8 + (i * 4)
		lump_column_offsets[i] = binary.LittleEndian.Uint32(w.raw_file[pos : pos+4])
	}

	img := image.NewRGBA(image.Rect(0, 0, int(sprite.Width), int(sprite.Height)))

	for i := 0; i < int(sprite.Width-1); i++ {
		offset := lump_column_offsets[i] + uint32(lump.Filepos)
		row_start := uint8(0)
		for row_start != 255 {
			row_start = w.raw_file[offset]
			offset += 1
			if row_start == 255 {
				break
			}
			length := uint8(w.raw_file[offset])
			offset += 1
			offset += 1
			data_slice := w.raw_file[offset : offset+uint32(length)]
			offset += uint32(length)
			for l := 0; l < int(length); l++ {
				pcolor := playpal.At(int(data_slice[l]), 0)
				img.Set(i, l+int(row_start), pcolor)
			}
			offset += 1
		}
	}
	return img
}
