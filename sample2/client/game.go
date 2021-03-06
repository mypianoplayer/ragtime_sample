package main

import (
	"image"
	"log"

	"azul3d.org/engine/gfx"
	"azul3d.org/engine/gfx/camera"
	"azul3d.org/engine/gfx/gfxutil"
	"azul3d.org/engine/gfx/window"
	"azul3d.org/engine/keyboard"
	"azul3d.org/engine/lmath"

	"azul3d.org/examples/abs"
)

type Game struct {
	cam     *camera.Camera
	event   chan window.Event
	rtColor *gfx.Texture
	card    *gfx.Object
	scene   *Scene
}

func NewGame() *Game {
	return &Game{
		scene: NewScene(),
	}
}

func (g *Game) Init(w window.Window, d gfx.Device) {

	// Create a new perspective (3D) camera.
	g.cam = camera.New(d.Bounds())

	// Move the camera back two units away from the card.
	g.cam.SetPos(lmath.Vec3{0, -2, 0})

	// Create a texture to hold the color data of our render-to-texture.
	g.rtColor = gfx.NewTexture()
	g.rtColor.MinFilter = gfx.LinearMipmapLinear
	g.rtColor.MagFilter = gfx.Linear

	// Choose a render to texture format.
	cfg := d.Info().RTTFormats.ChooseConfig(gfx.Precision{
		// We want 24/bpp RGB color buffer.
		RedBits: 8, GreenBits: 8, BlueBits: 8,

		// We could also request a depth or stencil buffer here, by simply
		// using the lines:
		// DepthBits: 24,
		// StencilBits: 24,
	}, true)

	// Print the configuration we chose.
	log.Printf("RTT ColorFormat=%v, DepthFormat=%v, StencilFormat=%v\n", cfg.ColorFormat, cfg.DepthFormat, cfg.StencilFormat)

	// Color buffer will go into our rtColor texture.
	cfg.Color = g.rtColor

	// We will render to a 512x512 area.
	cfg.Bounds = image.Rect(0, 0, 512, 512)

	// Create our render-to-texture canvas.
	rtCanvas := d.RenderToTexture(cfg)
	if rtCanvas == nil {
		// Important! Check if the canvas is nil. If it is their graphics
		// hardware doesn't support render to texture. Sorry!
		log.Fatal("Graphics hardware does not support render to texture.")
	}

	// Read the GLSL shaders from disk.
	shader, err := gfxutil.OpenShader(abs.Path("azul3d_rtt/rtt"))
	if err != nil {
		log.Fatal(err)
	}

	// Create a card mesh.
	cardMesh := gfx.NewMesh()
	cardMesh.Vertices = []gfx.Vec3{
		// Bottom-left triangle.
		{-1, 0, -1},
		{1, 0, -1},
		{-1, 0, 1},

		// Top-right triangle.
		{-1, 0, 1},
		{1, 0, -1},
		{1, 0, 1},
	}
	cardMesh.TexCoords = []gfx.TexCoordSet{
		{
			Slice: []gfx.TexCoord{
				{0, 1},
				{1, 1},
				{0, 0},

				{0, 0},
				{1, 1},
				{1, 0},
			},
		},
	}

	// Create a card object.
	g.card = gfx.NewObject()
	g.card.State = gfx.NewState()
	g.card.FaceCulling = gfx.NoFaceCulling
	g.card.AlphaMode = gfx.AlphaToCoverage
	g.card.Shader = shader
	g.card.Textures = []*gfx.Texture{g.rtColor}
	g.card.Meshes = []*gfx.Mesh{cardMesh}

	// Create an event mask for the events we are interested in.
	evMask := window.FramebufferResizedEvents
	evMask |= window.KeyboardTypedEvents

	// Create a channel of events.
	g.event = make(chan window.Event, 256)

	// Have the window notify our channel whenever events occur.
	w.Notify(g.event, evMask)

	// Draw some colored stripes onto the render to texture canvas. The result
	// is stored in the rtColor texture, and we can then display it on a card
	// below without even rendering the stripes every frame.
	stripeColor1 := gfx.Color{1, 0, 0, 1}   // red
	stripeColor2 := gfx.Color{1, 0.5, 1, 1} // green
	stripeWidth := 12                       // pixels
	flipColor := false
	b := rtCanvas.Bounds()
	for i := 0; (i * stripeWidth) < b.Dx(); i++ {
		flipColor = !flipColor
		x := i * stripeWidth
		dst := image.Rect(x, b.Min.Y, x+stripeWidth, b.Max.Y)
		if flipColor {
			rtCanvas.Clear(dst, stripeColor1)
		} else {
			rtCanvas.Clear(dst, stripeColor2)
		}
	}

	// Render the rtCanvas to the rtColor texture.
	rtCanvas.Render()
}

func (g *Game) Update(w window.Window, d gfx.Device) {

	// Handle each pending event.
	window.Poll(g.event, func(e window.Event) {
		switch ev := e.(type) {
		case window.FramebufferResized:
			// Update the camera's projection matrix for the new width and
			// height.
			g.cam.Update(d.Bounds())

		case keyboard.Typed:
			if ev.S == "m" || ev.S == "M" {
				// Toggle mipmapping.
				if g.rtColor.MinFilter == gfx.LinearMipmapLinear {
					g.rtColor.MinFilter = gfx.Linear
				} else {
					g.rtColor.MinFilter = gfx.LinearMipmapLinear
				}
			}
		}
	})

	// Rotate the card on the Z axis 15 degrees/sec.
	//		rot := card.Rot()
	//		card.SetRot(lmath.Vec3{
	//			X: rot.X,
	//			Y: rot.Y,
	//			Z: rot.Z + (15 * d.Clock().Dt()),
	//		})

	// Clear color and depth buffers.
	d.Clear(d.Bounds(), gfx.Color{1, 1, 1, 1})
	d.ClearDepth(d.Bounds(), 1.0)

	// Draw the card.
	d.Draw(d.Bounds(), g.card, g.cam)

	// Render the frame.
	d.Render()

}
