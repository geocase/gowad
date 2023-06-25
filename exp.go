package main

import (
	"fmt"
	"image/png"
	"os"
	"sort"

	"github.com/gonutz/wui/v2"
)

func main() {
	current_wad, _ := LoadWadFromPath("DOOM.WAD")

	play := current_wad.DecodePlaypal("PLAYPAL")
	img_file, _ := os.Create("PLAYPAL" + ".png")
	png.Encode(img_file, play)
	img_file.Close()

	img_target := "HELP1"
	pinky := current_wad.DecodeImage(img_target)
	img_file, _ = os.Create(img_target + ".png")
	png.Encode(img_file, pinky)
	img_file.Close()

	win := wui.NewWindow()
	win.SetTitle("Hello, world!")
	win.SetResizable(true)

	// split
	left_panel := wui.NewPanel()
	left_panel.SetBorderStyle(wui.PanelBorderRaised)
	win_x, win_y := win.Size()
	left_panel.SetBounds(0, 0, win_x/2, win_y)
	left_panel.SetAnchors(wui.AnchorMinAndCenter, wui.AnchorMinAndMax)

	right_panel := wui.NewPanel()
	right_panel.SetBorderStyle(wui.PanelBorderRaised)
	win_x, win_y = win.Size()
	right_panel.SetBounds(win_x/2, 0, win_x/2, win_y)
	right_panel.SetAnchors(wui.AnchorMaxAndCenter, wui.AnchorMinAndMax)

	selection := wui.NewStringList()
	left_panel.Add(selection)
	selection.SetBounds(left_panel.Bounds())
	selection.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)

	winimage := wui.NewImage(pinky)

	pb := wui.NewPaintBox()
	right_panel.Add(pb)
	pb.SetBounds(0, 0, win_x/2, win_y)
	pb.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)
	pb.SetOnPaint(func(c *wui.Canvas) {
		c.FillRect(0, 0, c.Width(), c.Height(), wui.RGB(255, 255, 255))
		c.DrawImage(winimage, wui.Rectangle{}, c.Width()/2-winimage.Width()/2, c.Height()/2-winimage.Height()/2)
	})

	selection.SetOnChange(func(i int) {
		if i >= 0 {
			target := selection.Items()[i]
			fmt.Println(target)
			winimage = wui.NewImage(current_wad.DecodeImage(target))
			pb.Paint()
		}
	})

	keys := make([]string, 0, len(current_wad.directory))
	for k := range current_wad.directory {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		selection.AddItem(k)
	}

	win.Add(left_panel)
	win.Add(right_panel)
	win.Show()
}
