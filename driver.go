package main

import (
	"image/png"
	"log"
	"os"
	"wadlib/wad"

	"github.com/go-audio/wav"
)

// greyscale resample 80 - 113
func main() {
	current_wad, _ := wad.Load("DOOM1.WAD", true)
	img_target := "PLAYA1"
	pp := current_wad.Lump("PLAYPAL").AsPlaypal()
	playa := current_wad.Lump(img_target).AsSprite(pp)
	img_file, _ := os.Create(img_target + ".png")
	png.Encode(img_file, playa)
	img_file.Close()

	sfx_target := "DSPLPAIN"
	sfx := current_wad.Lump(sfx_target).AsDMXSound()
	_ = sfx
	wav_out, _ := os.Create(sfx_target + ".wav")
	wav_encoder := wav.NewEncoder(wav_out, sfx.Format.SampleRate, sfx.SourceBitDepth, 1, 1)
	err := wav_encoder.Write(sfx)
	if err != nil {
		log.Fatal(err)
	}
	wav_encoder.Close()

	new_wad := wad.New()
	switch_sfx := current_wad.Lump(sfx_target)
	switch_sfx.SetName("DSPISTOL")
	new_wad.AddLump(switch_sfx)
	help_gfx := current_wad.Lump("CHGGA0")
	help_gfx.SetName("PISGA0")
	new_wad.AddLump(help_gfx)

	wad_out, _ := os.Create("out.wad")
	new_wad.Write(wad_out)
	wad_out.Close()
}
