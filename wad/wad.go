package wad

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"strings"

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
	wadinfo  wadinfo_t
	lumps    map[string]rawlump_t
	raw_file []byte
}

const (
	LUMP_HINT_UNDEF  = iota
	LUMP_HINT_SPRITE = iota
)

type rawlump_t struct {
	name      [8]byte
	size      int32
	type_hint int
	data      []byte
}

func (rl *rawlump_t) SetName(n string) {
	copy(rl.name[:], n)
}

func New() wadfile_t {
	id := [4]byte{'P', 'W', 'A', 'D'}
	w := wadfile_t{
		wadinfo: wadinfo_t{
			Identification: id,
			Numlumps:       0,
			Infotableofs:   0,
		},
		lumps: make(map[string]rawlump_t),
	}

	return w
}

func (wf wadfile_t) Write(w io.WriteSeeker) {
	// organize data
	sprites := make([]rawlump_t, 0)
	undefs := make([]rawlump_t, 0)
	directory := make([]filelump_t, 0)

	sprites = append(sprites, rawlump_t{
		name:      [8]byte{'S', '_', 'S', 'T', 'A', 'R', 'T'},
		size:      0,
		type_hint: LUMP_HINT_UNDEF,
		data:      nil,
	})

	for _, l := range wf.lumps {
		switch l.type_hint {
		case LUMP_HINT_UNDEF:
			undefs = append(undefs, l)
		case LUMP_HINT_SPRITE:
			sprites = append(sprites, l)
		}
	}

	sprites = append(sprites, rawlump_t{
		name:      [8]byte{'S', '_', 'E', 'N', 'D', 0, 0, 0},
		size:      0,
		type_hint: LUMP_HINT_UNDEF,
		data:      nil,
	})

	offset := 12 // 4 bytes, 2 int32s = 12 bytes
	w.Seek(int64(offset), 0)

	for _, u := range undefs {
		binary.Write(w, binary.LittleEndian, u.data)
		directory = append(directory, filelump_t{Filepos: int32(offset),
			Size: u.size,
			Name: u.name})
		offset += int(u.size)
	}

	for _, i := range sprites {
		if i.size != 0 {
			binary.Write(w, binary.LittleEndian, i.data)
		}
		directory = append(directory, filelump_t{Filepos: int32(offset),
			Size: i.size,
			Name: i.name})
		offset += int(i.size)
	}
	wf.wadinfo.Numlumps = int32(len(undefs) + len(sprites))
	wf.wadinfo.Infotableofs = int32(offset)
	for _, dir := range directory {
		binary.Write(w, binary.LittleEndian, dir)
		fmt.Println(string(dir.Name[:]))
	}

	w.Seek(0, 0)
	binary.Write(w, binary.LittleEndian, wf.wadinfo)

}

func Load(path string, ignore_map_data_lumps bool) (wadfile_t, error) {
	// TODO: load wad into memory once and then perform actions on buffer
	// TODO: decide how to store map lumps
	// TODO: Tag lumps according to special pointer lumps
	w := wadfile_t{}
	w.wadinfo, _ = LoadWadHeader(path)

	raw_dir := make([]filelump_t, w.wadinfo.Numlumps)
	w.lumps = make(map[string]rawlump_t)
	f, _ := os.Open(path)
	f.Seek(int64(w.wadinfo.Infotableofs), 0)
	binary.Read(f, binary.LittleEndian, &raw_dir)

	map_lump_names := []string{"THINGS", "LINEDEFS", "SIDEDEFS", "VERTEXES", "SEGS", "SSECTORS", "NODES", "SECTORS", "REJECT", "BLOCKMAP"} // THIS ORDER IS IMPORTANT
	type_hint := LUMP_HINT_UNDEF
	for i := 0; i < len(raw_dir); i++ {
		clean := strings.Trim(string(raw_dir[i].Name[:]), string(0))
		if clean == "S_START" {
			type_hint = LUMP_HINT_SPRITE
		}
		if clean == "S_END" {
			type_hint = LUMP_HINT_UNDEF
		}

		if (clean[0] == 'E' && clean[2] == 'M') ||
			(clean[0] == 'M' && clean[1] == 'A' && clean[2] == 'P') {
			i += 1
			if ignore_map_data_lumps {
				// load data into unique map lump struct
				for _, s := range map_lump_names {
					clean = strings.Trim(string(raw_dir[i].Name[:]), string(0))
					// TODO: this
					_ = s
					i += 1
				}
			}
		} else {
			new_lump := rawlump_t{name: raw_dir[i].Name, size: raw_dir[i].Size, type_hint: type_hint}
			f.Seek(int64(raw_dir[i].Filepos), 0)
			new_lump.data = make([]byte, raw_dir[i].Size)
			binary.Read(f, binary.LittleEndian, new_lump.data)
			w.lumps[clean] = new_lump
		}
	}
	f.Seek(0, 0)
	file_stat, _ := f.Stat()
	file_size := file_stat.Size()
	w.raw_file = make([]byte, file_size)
	binary.Read(f, binary.LittleEndian, &w.raw_file)
	f.Close()
	return w, nil
}

func (w wadfile_t) Lump(key string) rawlump_t {
	return w.lumps[key]
}

func (wf *wadfile_t) AddLump(rl rawlump_t) {
	clean := strings.Trim(string(rl.name[:]), string(0))
	wf.lumps[clean] = rl
	wf.wadinfo.Numlumps = int32(len(wf.lumps))
}

func (l *rawlump_t) SetTypeHint(hint int) {
	l.type_hint = hint
}

func (l rawlump_t) AsPlaypal() *image.RGBA {
	ret := image.NewRGBA(image.Rect(0, 0, 256, 1))
	for x := 0; x < len(l.data)/3; x++ {
		i := x * 3
		ret.Set(x, 0, color.NRGBA{
			R: l.data[i+0],
			G: l.data[i+1],
			B: l.data[i+2],
			A: 255,
		})
	}
	return ret
}

func (l rawlump_t) AsSprite(playpal *image.RGBA) *image.RGBA {
	sprite := sprite_lump_t{}
	sprite.Width = binary.LittleEndian.Uint16(l.data[0:2])
	sprite.Height = binary.LittleEndian.Uint16(l.data[2:4])
	lump_column_offsets := make([]uint32, sprite.Width)
	for i := 0; i < int(sprite.Width); i++ {
		pos := 8 + (i * 4)
		lump_column_offsets[i] = binary.LittleEndian.Uint32(l.data[pos : pos+4])
	}

	img := image.NewRGBA(image.Rect(0, 0, int(sprite.Width), int(sprite.Height)))

	for i := 0; i < int(sprite.Width-1); i++ {
		offset := lump_column_offsets[i]
		row_start := uint8(0)
		for row_start != 255 {
			row_start = l.data[offset]
			offset += 1
			if row_start == 255 {
				break
			}
			length := uint8(l.data[offset])
			offset += 1
			offset += 1
			data_slice := l.data[offset : offset+uint32(length)]
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

func (l rawlump_t) AsDMXSound() *audio.IntBuffer {
	dmx_lump := dmx_lump_t{
		Format:      binary.LittleEndian.Uint16(l.data[0:2]),
		Sample_rate: binary.LittleEndian.Uint16(l.data[2:4]),
		Sample_cnt:  binary.LittleEndian.Uint32(l.data[4:8]),
	}
	dmx_lump.Data = make([]uint8, dmx_lump.Sample_cnt)
	copy(dmx_lump.Data[:], l.data[8:8+int32(dmx_lump.Sample_cnt)])
	ret := audio.IntBuffer{
		SourceBitDepth: 8,
		Format:         &audio.Format{NumChannels: 1, SampleRate: int(dmx_lump.Sample_rate)},
	}
	ret.Data = make([]int, dmx_lump.Sample_cnt)
	for i := 0; i < len(ret.Data); i++ {
		ret.Data[i] = int(dmx_lump.Data[i])
	}

	return &ret
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
