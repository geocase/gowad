package wad

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"os"
	"strings"
	"fmt"

	"github.com/go-audio/audio"
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

type dmx_lump_t struct {
	Format      uint16
	Sample_rate uint16
	Sample_cnt  uint32
	Data        []uint8
}

type wadfile_t struct {
	wadinfo   wadinfo_t
	directory map[string]filelump_t
	raw_file  []byte
}

func (w wadfile_t) Test(key string) {
	fmt.Println(key)
	for i := range w.directory {
		if  strings.Trim(string(i), string(0)) == key {
			fmt.Println(i)
		}
	}
}

func LoadWadFromPath(path string, ignore_map_data_lumps bool) (wadfile_t, error) {
	// TODO: load wad into memory once and then perform actions on buffer
	// TODO: decide how to store map lumps
	w := wadfile_t{}
	w.wadinfo, _ = LoadWadHeader(path)

	raw_dir := make([]filelump_t, w.wadinfo.Numlumps)
	w.directory = make(map[string]filelump_t)
	// playpal_lump_position := filelump_t{}
	f, _ := os.Open(path)
	f.Seek(int64(w.wadinfo.Infotableofs), 0)
	binary.Read(f, binary.LittleEndian, &raw_dir)

	map_lump_names := []string{"THINGS", "LINEDEFS", "SIDEDEFS", "VERTEXES", "SEGS", "SSECTORS", "NODES", "SECTORS", "REJECT", "BLOCKMAP",} // THIS ORDER IS IMPORTANT

	for i := 0; i < len(raw_dir); i++ {
		clean := strings.Trim(string(raw_dir[i].Name[:]), string(0))
		if (clean[0] == 'E' && clean[2] == 'M') || 
		(clean[0] == 'M' && clean[1] == 'A' && clean[2] == 'P') {
			fmt.Println(clean)
			i += 1
			if(ignore_map_data_lumps) {
				// load data into unique map lump struct
				for _, s := range map_lump_names {
					clean = strings.Trim(string(raw_dir[i].Name[:]), string(0))
					fmt.Println(clean, " ", s == clean)
					i += 1
				}
			}
		} else {
			w.directory[clean] = raw_dir[i]
		}
	}
	f.Seek(0, 0)
	file_stat, _ := f.Stat()
	file_size := file_stat.Size()
	w.raw_file = make([]byte, file_size)
	binary.Read(f, binary.LittleEndian, &w.raw_file)
	f.Close()
	if ignore_map_data_lumps {
		for _, n := range map_lump_names {
			delete(w.directory, n)
		}
	}
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

func (w wadfile_t) DecodeSound(key string) audio.IntBuffer {
	audio_lump_info := w.directory[key]

	dmx_lump := dmx_lump_t{
		Format:      binary.LittleEndian.Uint16(w.raw_file[audio_lump_info.Filepos : audio_lump_info.Filepos+2]),
		Sample_rate: binary.LittleEndian.Uint16(w.raw_file[audio_lump_info.Filepos+2 : audio_lump_info.Filepos+4]),
		Sample_cnt:  binary.LittleEndian.Uint32(w.raw_file[audio_lump_info.Filepos+4 : audio_lump_info.Filepos+8]),
	}
	dmx_lump.Data = make([]uint8, dmx_lump.Sample_cnt)
	copy(dmx_lump.Data[:], w.raw_file[audio_lump_info.Filepos+8:audio_lump_info.Filepos+8+int32(dmx_lump.Sample_cnt)])
	ret := audio.IntBuffer{
		SourceBitDepth: 8,
		Format:         &audio.Format{NumChannels: 1, SampleRate: int(dmx_lump.Sample_rate)},
	}
	ret.Data = make([]int, dmx_lump.Sample_cnt)
	for i := 0; i < len(ret.Data); i++ {
		ret.Data[i] = int(dmx_lump.Data[i])
	}

	return ret
}
