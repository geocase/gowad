package main

import (
	"image/png"
	"log"
	"os"

	"github.com/go-audio/wav"
)

func main() {
	current_wad, _ := LoadWadFromPath("DOOM.WAD")

	img_target := "PLAYA1"
	pinky := current_wad.DecodeImage(img_target)
	img_file, _ := os.Create(img_target + ".png")
	png.Encode(img_file, pinky)
	img_file.Close()

	sfx_target := "DSPLPAIN"

	sfx := current_wad.DecodeSound(sfx_target)
	_ = sfx
	wav_out, _ := os.Create(sfx_target + ".wav")
	wav_encoder := wav.NewEncoder(wav_out, sfx.Format.SampleRate, sfx.SourceBitDepth, 1, 1)
	err := wav_encoder.Write(&sfx)
	if err != nil {
		log.Fatal(err)
	}
	wav_encoder.Close()
}
